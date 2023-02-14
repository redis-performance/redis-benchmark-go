# redis-benchmark-go

## Overview

This repo contains code to mimic redis-benchmark capabilities in go. 


## Getting Started

### Download Standalone binaries ( no Golang needed )

If you don't have go on your machine and just want to use the produced binaries you can download the following prebuilt bins:

https://github.com/redis-performance/redis-benchmark-go/releases/latest

Here's how: 

**Linux**

x86
```
wget -c https://github.com/redis-performance/redis-benchmark-go/releases/latest/download/redis-benchmark-go-linux-amd64.tar.gz -O - | tar -xz

# give it a try
./redis-benchmark-go --help
```

arm64
```
wget -c https://github.com/redis-performance/redis-benchmark-go/releases/latest/download/redis-benchmark-go-linux-arm64.tar.gz -O - | tar -xz

# give it a try
./redis-benchmark-go --help
```

**OSX**

x86
```
wget -c https://github.com/redis-performance/redis-benchmark-go/releases/latest/download/redis-benchmark-go-darwin-amd64.tar.gz -O - | tar -xz

# give it a try
./redis-benchmark-go --help
```

arm64
```
wget -c https://github.com/redis-performance/redis-benchmark-go/releases/latest/download/redis-benchmark-go-darwin-arm64.tar.gz -O - | tar -xz

# give it a try
./redis-benchmark-go --help
```

**Windows**
```
wget -c https://github.com/redis-performance/redis-benchmark-go/releases/latest/download/redis-benchmark-go-windows-amd64.tar.gz -O - | tar -xz

# give it a try
./redis-benchmark-go --help
```

### Installation in a Golang env

The easiest way to get and install the benchmark utility with a Go Env is to use
`go get` and then `go install`:
```bash
# Fetch this repo
go get github.com/redis-performance/redis-benchmark-go
cd $GOPATH/src/github.com/redis-performance/redis-benchmark-go
make
```

## Usage of redis-benchmark-go

```
$ redis-benchmark-go --help
Usage of redis-benchmark-go:
  -a string
    	Password for Redis Auth.
  -c uint
    	number of clients. (default 50)
  -d uint
    	Data size of the expanded string __data__ value in bytes. The benchmark will expand the string __data__ inside an argument with a charset with length specified by this parameter. The substitution changes every time a command is executed. (default 3)
  -debug int
    	Client debug level.
  -h string
    	Server hostname. (default "127.0.0.1")
  -l	Loop. Run the tests forever.
  -multi
    	Run each command in multi-exec.
  -n uint
    	Total number of requests (default 10000000)
  -oss-cluster
    	Enable OSS cluster mode.
  -p int
    	Server port. (default 12000)
  -r uint
    	keyspace length. The benchmark will expand the string __key__ inside an argument with a number in the specified range from 0 to keyspacelen-1. The substitution changes every time a command is executed. (default 1000000)
  -random-seed int
    	random seed to be used. (default 12345)
  -resp int
    	redis command response protocol (2 - RESP 2, 3 - RESP 3) (default 2)
  -rps int
    	Max rps. If 0 no limit is applied and the DB is stressed up to maximum.
  -v	Output version and exit
  -wait-replicas int
    	If larger than 0 will wait for the specified number of replicas.
  -wait-replicas-timeout-ms int
    	WAIT timeout when used together with -wait-replicas. (default 1000)
```

## Sample output - Rate limited example. 1000 Keys, 100K commands, @10K RPS

```
$ redis-benchmark-go -r 1000 -n 100000 --rps 10000  hset __key__ f1 __data__
Total clients: 50. Commands per client: 2000 Total commands: 100000
Using random seed: 12345
                 Test time                    Total Commands              Total Errors                      Command Rate           p50 lat. (msec)
                        9s [100.0%]                    100000                       282 [0.3%]                   9930.65                      0.22	
#################################################
Total Duration 9.001 Seconds
Total Errors 282
Throughput summary: 11110 requests per second
Latency summary (msec):
          p50       p95       p99
        0.224     0.676     1.501
```

## Sample output - Rate limited SET + WAIT example. 39M Keys, 500K commands, @5K RPS

```
$ redis-benchmark-go -p 6379 -r 39000000 -n 500000 -wait-replicas 1 -wait-replicas-timeout-ms 500 --rps 5000 SET __key__ __data__
IPs [127.0.0.1]
Total clients: 50. Commands per client: 10000 Total commands: 500000
Using random seed: 12345
                 Test time                    Total Commands              Total Errors                      Command Rate           p50 lat. (msec)
                       99s [100.0%]                    500000                         0 [0.0%]                   2574.08                      0.28      
#################################################
Total Duration 99.000 Seconds
Total Errors 0
Throughput summary: 5051 requests per second
Latency summary (msec):
          avg       p50       p95       p99
        0.377     0.275     0.703     2.459
```

## Sample output - 10M commands

```
$ redis-benchmark-go -p 20000 --debug 1  hset __key__ f1 __data__
  Total clients: 50. Commands per client: 200000 Total commands: 10000000
  Using random seed: 12345
                   Test time                    Total Commands              Total Errors                      Command Rate           p50 lat. (msec)
                         42s [100.0%]                  10000000                         0 [0.0%]                 172737.59                      0.17      
  #################################################
  Total Duration 42.000 Seconds
  Total Errors 0
  Throughput summary: 238094 requests per second
  Latency summary (msec):
            p50       p95       p99
          0.168     0.403     0.528
```


## Sample output - running in loop mode ( Ctrl+c to stop )

```
$ redis-benchmark-go -p 20000 --debug 1 -l  hset __key__ f1 __data__
Running in loop until you hit Ctrl+C
Using random seed: 12345
                 Test time                    Total Commands              Total Errors                      Command Rate           p50 lat. (msec)
^C                     10s [----%]                   2788844                         0 [0.0%]                 254648.64                      0.16       
received Ctrl-c - shutting down

#################################################
Total Duration 10.923 Seconds
Total Errors 0
Throughput summary: 274843 requests per second
Latency summary (msec):
          p50       p95       p99
        0.162     0.372     0.460
```