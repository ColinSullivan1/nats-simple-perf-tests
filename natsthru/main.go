// Copyright 2012-2022 Colin Sullivan
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
)

func usage() {
	log.Printf("Usage: natsthru [-s server] [-creds file] <send|recv> <subject> <payload size>\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func main() {
	var urls = flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma)")
	var userCreds = flag.String("creds", "", "User Credentials File")
	var tlsClientCert = flag.String("tlscert", "", "TLS client certificate file")
	var tlsClientKey = flag.String("tlskey", "", "Private key file for client certificate")
	var tlsCACert = flag.String("tlscacert", "", "CA certificate to verify peer against")
	var showHelp = flag.Bool("h", false, "Show help message")
	var window = flag.Int("window", 1024, "Number of messages to window between requests")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		showUsageAndExit(1)
	}

	// Connect Options.
	opts := []nats.Option{nats.Name("NATS Max Throughput Test")}

	// Use UserCredentials
	if *userCreds != "" {
		opts = append(opts, nats.UserCredentials(*userCreds))
	}

	// Use TLS client authentication
	if *tlsClientCert != "" && *tlsClientKey != "" {
		opts = append(opts, nats.ClientCert(*tlsClientCert, *tlsClientKey))
	}

	// Use specific CA certificate
	if *tlsCACert != "" {
		opts = append(opts, nats.RootCAs(*tlsCACert))
	}

	action, subj := args[0], args[1]
	if action != "send" && action != "recv" {
		log.Printf("invalid action: %s", action)
		showUsageAndExit(1)
	}

	// Connect to NATS
	nc, err := nats.Connect(*urls, opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	log.Printf("Connected to server %s\n", nc.ConnectedServerName())
	if action == "send" {
		size, err := strconv.Atoi(args[2])
		if err != nil {
			showUsageAndExit(1)
		}
		sendMsgs(nc, subj, size, *window)
	} else if action == "recv" {
		recvMsgs(nc, subj)
	}

	nc.Flush()
}

func recvMsgs(nc *nats.Conn, subject string) {
	sub, err := nc.SubscribeSync(subject)
	if err != nil {
		log.Printf("couldn't subscribe: %v", err)
		os.Exit(1)
	}
	log.Printf("Subscribed to %q and waiting for messages.", subject)

	i := 0
	lastPrint := time.Now()
	for {
		msg, err := sub.NextMsg(time.Hour)
		if err != nil {
			log.Printf("couldn't retrieve next message: %v", err)
			os.Exit(1)
		}
		if msg == nil {
			log.Printf("Timeout.  Exiting.\n")
			os.Exit(1)
		}
		i++
		if msg.Reply != "" {
			// Send back the pending message count, indicating how
			// backed up the subscriber is.
			pmsgs, _, err := sub.Pending()
			if err != nil {
				log.Printf("couldn't get pending msg count: %v", err)
			}
			payload := make([]byte, 8)
			binary.LittleEndian.PutUint64(payload[0:], uint64(pmsgs))
			msg.Respond(payload)

			if time.Since(lastPrint) > time.Second*3 {
				log.Printf("Received %d messages.", i)
				lastPrint = time.Now()
			}
		}
	}
}

func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

func sendMsgs(nc *nats.Conn, subject string, size, window int) {
	payload := make([]byte, size)
	inbox := nats.NewInbox()
	waitChan := make(chan int)
	wnd := int32(window)

	_, err := nc.Subscribe(inbox, func(msg *nats.Msg) {
		if msg.Data == nil {
			return
		}
		if msg.Header.Get("Status") == "503" {
			log.Println("No service available - start the receiver. Exiting")
			os.Exit(1)
		}

		// Not aggressive here - slowly increase the outstanding
		// msg window if the sub is keeping up.  We want to
		// avoid the sawtooth pattern and keep the pipe always full.
		if binary.LittleEndian.Uint64(msg.Data) == 0 {
			w := atomic.LoadInt32(&wnd)
			if w < 50*1024 {
				atomic.AddInt32(&wnd, int32(window))
			}
		}
		// signal the publisher
		waitChan <- 1
	})
	if err != nil {
		fmt.Printf("couldn't subscribe: %v", err)
	}

	var requestInProgress bool
	var i int
	var outstanding int
	start := time.Now()
	lastPrint := time.Now()
	for {
		nc.Publish(subject, payload)
		i++

		w := int(atomic.LoadInt32(&wnd))
		if i%w == 0 && !requestInProgress {
			if err := nc.PublishRequest(subject, inbox, payload); err != nil {
				if err == nats.ErrNoResponders {
					log.Printf("No responders.\n")
				}
			}
			requestInProgress = true
		}

		if requestInProgress {
			outstanding++
			if outstanding > w {
				select {
				case <-waitChan:
				case <-time.After(2 * time.Second):
					log.Printf("Timeout waiting for receiver response.")
					// Agressively shrink our window back down.
					// TODO (cls) channel could have late entry, add request seqno.
					atomic.StoreInt32(&wnd, int32(w/2))
				}

				outstanding = 0
				requestInProgress = false
				if time.Since(lastPrint) > time.Second*3 {
					mps := float64(i) / time.Since(start).Seconds()
					bps := int64(mps) * int64(size)
					fmt.Printf("Window: %d, Rate: %d msgs/sec, %s bytes/sec\n", w, int(mps), ByteCountIEC(bps))
					lastPrint = time.Now()
				}
			}
		}
	}
}
