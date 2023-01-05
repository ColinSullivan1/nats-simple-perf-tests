#!/bin/bash

# Run latencies between all sets of servers
if [ "$2" ]; then
  echo "running latency tests between $1 and $2"
else
  echo "usage:  run_latency.sh servera serverb"
  exit
fi

mkdir -p 2>/dev/null results

MSGSZ=128

nats latency --server="nats://$1:4222" --server-b="$2:4223" --size $MSGSZ --rate 10     --duration 30s --histogram=results/$1_$2_512_10_30s
nats latency --server="nats://$1:4222" --server-b="$2:4223" --size $MSGSZ --rate 100    --duration 30s --histogram=results/$1_$2_512_100_30s
nats latency --server="nats://$1:4222" --server-b="$2:4223" --size $MSGSZ --rate 1000   --duration 30s --histogram=results/$1_$2_512_1000_30s
nats latency --server="nats://$1:4222" --server-b="$2:4223" --size $MSGSZ --rate 10000  --duration 30s --histogram=results/$1_$2_512_10000_30s
nats latency --server="nats://$1:4222" --server-b="$2:4223" --size $MSGSZ --rate 100000 --duration 30s --histogram=results/$1_$2_512_100000_30s
