#!/bin/bash

mkdir tmp 2>/dev/null
cd tmp

curl -sLJO https://github.com/nats-io/natscli/releases/download/v0.0.35/nats-0.0.35-linux-amd64.zip
unzip nats-0.0.35-linux-amd64.zip 1>/dev/null
cp nats-0.0.35-linux-amd64/nats ..

curl -sLJO https://github.com/nats-io/nats-top/releases/download/v0.5.3/nats-top_0.5.3_darwin_amd64.tar.gz
tar -xvf nats-top_0.5.3_darwin_amd64.tar.gz 2>/dev/null
cp nats-top ..

cd ..
rm -rf tmp