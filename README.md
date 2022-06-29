# redis-benchmark-go

## Overview

A constant throughput, correct latency recording variant of [redis-benchmark](https://github.com/redis/redis/blob/unstable/src/redis-benchmark.c) and [memtier_benchmark](https://github.com/RedisLabs/memtier_benchmark/)

## Getting started with docker

### Rate limited example. 1000 Keys, 100K commands, @10K RPS

```bash
docker run --network=host codeperf/redis-benchmark-go:unstable -r 1000 -n 100000 --rps 10000  hset __key__ f1 __data__
```

### Rate limited example. SET + WAIT example. 1M Keys, 500K commands, @5K RPS

```bash
docker run --network=host codeperf/redis-benchmark-go:unstable -r 1000000 -n 500000 -wait-replicas 1 -wait-replicas-timeout-ms 500 --rps 5000 SET __key__ __data__
```

## Getting Started with prebuilt standalone binaries ( no Golang needed )

If you don't have go on your machine and just want to use the produced binaries you can download the following prebuilt bins:

| OS | Arch | Link |
| :---         |     :---:      |          ---: |
| Linux   | amd64  (64-bit X86)     | [redis-benchmark-go-v0.1.0-linux-amd64.tar.gz](https://github.com/filipecosta90/redis-benchmark-go/releases/download/v0.1.0/redis-benchmark-go-v0.1.0-linux-amd64.tar.gz)    |
| Linux   | arm64 (64-bit ARM)     | [redis-benchmark-go-v0.1.0-linux-arm64.tar.gz](https://github.com/filipecosta90/redis-benchmark-go/releases/download/v0.1.0/redis-benchmark-go-v0.1.0-linux-arm64.tar.gz)    |
| Darwin   | amd64  (64-bit X86)     | [redis-benchmark-go-v0.1.0-darwin-amd64.tar.gz](https://github.com/filipecosta90/redis-benchmark-go/releases/download/v0.1.0/redis-benchmark-go-v0.1.0-darwin-amd64.tar.gz)    |
| Darwin   | arm64 (64-bit ARM)     | [redis-benchmark-go-v0.1.0-darwin-arm64.tar.gz](https://github.com/filipecosta90/redis-benchmark-go/releases/download/v0.1.0/redis-benchmark-go-v0.1.0-darwin-arm64.tar.gz)    |


Here's an example on how to use the above links for a linux based amd64 machine:

```bash
# Fetch it 
wget -c https://github.com/filipecosta90/redis-benchmark-go/releases/download/v0.1.0/redis-benchmark-go-v0.1.0-linux-amd64.tar.gz -O - | tar -xz

# give it a try 
./redis-benchmark-go --help
```

## Getting Started building from source

### Installing
This benchmark go program is **know to be supported for go >= 1.17**.
The easiest way to get and install the benchmark Go programs is to use `go get` and then `go install`:

```
go get github.com/filipecosta90/redis-benchmark-go
cd $GOPATH/src/github.com/filipecosta90/redis-benchmark-go
make
```
