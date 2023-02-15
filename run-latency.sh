#!/bin/bash

# Run latencies between all sets of servers
if [ "$2" ]; then
  echo "running latency tests between $1 and $2"
else
  echo "usage:  run_latency.sh servera serverb"
  exit
fi

mkdir -p 2>/dev/null results

MSGSZ=1024
DUR=1m
GOGC=off

echo "NATS Latency Test"
echo "Hostname: `hostname`"
echo "Server a: $1"
echo "Server b: $2"
echo ""
echo "test1"
sudo nice -n -20 ./nats latency --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 10     --duration $DUR --histogram=results/$1_$2_1024_10_$DUR
echo ""
echo "test2"
sudo nice -n -20 ./nats latency --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 100    --duration $DUR --histogram=results/$1_$2_1024_100_$DUR
echo ""
echo "test3"
sudo nice -n -20 ./nats latency --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 1000   --duration $DUR --histogram=results/$1_$2_1024_1000_$DUR
echo ""
echo "test4"
sudo nice -n -20 ./nats latency --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 10000  --duration $DUR --histogram=results/$1_$2_1024_10000_$DUR
echo ""
echo "test5"
sudo nice -n -20 ./nats latency --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 100000 --duration $DUR --histogram=results/$1_$2_1024_100000_$DUR
echo ""
echo "test6"
sudo nice -n -20 ./nats latency  --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 350000 --duration $DUR --histogram=results/$1_$2_1024_350000_$DUR
echo ""
echo "test7"
sudo nice -n -20 ./nats latency  --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 400000 --duration $DUR --histogram=results/$1_$2_1024_400000_$DUR
echo "test8"
sudo nice -n -20 ./nats latency  --creds sys.creds --server="$1" --server-b="$2" --size $MSGSZ --rate 800000 --duration $DUR --histogram=results/$1_$2_1024_800000_$DUR

