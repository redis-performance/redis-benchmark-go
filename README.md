# redis-benchmark-go

[![codecov](https://codecov.io/gh/redis-performance/redis-benchmark-go/graph/badge.svg?token=5B3WJTDAWP)](https://codecov.io/gh/redis-performance/redis-benchmark-go)

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

# Client side Caching benchmark

Client side caching was introduced in [version v1.0.0](https://github.com/redis-performance/redis-benchmark-go/releases/tag/v1.0.0) of this tool and requires the usage of the rueidis vanilla client.
This means that for using CSC you need to use a minimum of 2 extra flags on your benchmark, namely `-rueidis -csc`.

Bellow you can find all flags that control CSC behaviour:

```
  -csc
        Enable client side caching
  -csc-per-client-bytes int
        client side cache size that bind to each TCP connection to a single redis instance (default 134217728)
  -csc-ttl duration
        Client side cache ttl for cached entries (default 1m0s)
  -rueidis
        Use rueidis as the vanilla underlying client.
```

If you take the following benchmark command
```
$ ./redis-benchmark-go -rueidis -csc -n 2 -r 1 -c 1  -p 6379 GET key
```


The above example will send the following command to redis in case of cache miss:

```
//  CLIENT CACHING YES
//  MULTI
//  PTTL k
//  GET k
//  EXEC
```
If the key's TTL on the server is smaller than the client side TTL, the client side TTL will be capped.

On the second command execution for the same client, the command won't be issued to the server as visible bellow on the CSC Hits/sec column.


```
$ ./redis-benchmark-go -rueidis -csc -n 2 -r 1 -c 1  -p 6379 GET key
IPs [127.0.0.1]
Total clients: 1. Commands per client: 2 Total commands: 2
Using random seed: 12345
                 Test time                    Total Commands              Total Errors                      Command Rate              CSC Hits/sec     CSC Invalidations/sec           p50 lat. (msec)
                        0s [100.0%]                         2                         0 [0.0%]                         2                         1                         0                     0.002  
#################################################
Total Duration 0.000 Seconds
Total Errors 0
Throughput summary: 19218 requests per second
                    9609 CSC Hits per second
                    0 CSC Evicts per second
Latency summary (msec):
          avg       p50       p95       p99
        0.379     0.002     0.756     0.756

```

and as visible by the following server side monitoring during the above benchmark.

```
$ redis-cli monitor
OK
1695911011.777347 [0 127.0.0.1:56574] "HELLO" "3"
1695911011.777366 [0 127.0.0.1:56574] "CLIENT" "TRACKING" "ON" "OPTIN"
1695911011.777738 [0 127.0.0.1:56574] "CLIENT" "CACHING" "YES"
1695911011.777748 [0 127.0.0.1:56574] "MULTI"
1695911011.777759 [0 127.0.0.1:56574] "PTTL" "key"
1695911011.777768 [0 127.0.0.1:56574] "GET" "key"
1695911011.777772 [0 127.0.0.1:56574] "EXEC"
```

When a key is modified by some client, or is evicted because it has an associated expire time, 
or evicted because of a maxmemory policy, all the clients with tracking enabled that may have the key cached,
are notified with an invalidation message.

This can represent a large ammount of invalidation messages per second going through redis in each second. 
On the sample benchmark bellow, with 50 clients, doing 5% WRITES and 95% READS on a keyspace length of 10000 Keys, 
we've observed more than 50K invalidation messages per second and only 20K CSC Hits per second even on this read-heavy scenario. 

The goal of this CSC measurement capacibility is to precisely help you understand the do's and dont's on CSC and when it's best to use or avoid it. 

```
$ ./redis-benchmark-go -p 6379  -rueidis -r 10000 -csc -cmd "SET __key__ __data__" -cmd-ratio 0.05 -cmd "GET __key__" -cmd-ratio 0.95 --json-out-file results.json
IPs [127.0.0.1]
Total clients: 50. Commands per client: 200000 Total commands: 10000000
Using random seed: 12345
                 Test time                    Total Commands              Total Errors                      Command Rate              CSC Hits/sec     CSC Invalidations/sec           p50 lat. (msec)
                      125s [100.0%]                  10000000                         0 [0.0%]                     25931                      9842                     16777                     0.611  
#################################################
Total Duration 125.002 Seconds
Total Errors 0
Throughput summary: 79999 requests per second
                    20651 CSC Hits per second
                    54272 CSC Evicts per second
Latency summary (msec):
          avg       p50       p95       p99
        0.620     0.611     1.461     2.011
2023/09/28 15:36:13 Saving JSON results file to results.json
```

Eviction tracking and overhead. 
In case of invalidation of the key by other client, redis will send invalidation messages that  
