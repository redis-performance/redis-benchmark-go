# redis-benchmark-go

## Overview

This repo contains code to mimic redis-benchmark capabilities in go solely for OSS redis cluster. 


## Standalone binaries ( no Golang needed )

If you don't have go on your machine and just want to use the produced binaries you can download the following prebuilt bins:

| OS | Arch | Link |
| :---         |     :---:      |          ---: |
| Windows   | amd64  (64-bit X86)     | [redis-benchmark-go_windows_amd64.exe](https://s3.amazonaws.com/benchmarks.redislabs/tools/redis-benchmark-go/redis-benchmark-go_windows_amd64.exe)    |
| Linux   | amd64  (64-bit X86)     | [redis-benchmark-go_linux_amd64](https://s3.amazonaws.com/benchmarks.redislabs/tools/redis-benchmark-go/redis-benchmark-go_linux_amd64)    |
| Linux   | arm64 (64-bit ARM)     | [redis-benchmark-go_linux_arm64](https://s3.amazonaws.com/benchmarks.redislabs/tools/redis-benchmark-go/redis-benchmark-go_linux_arm64)    |
| Linux   | arm (32-bit ARM)    | [redis-benchmark-go_linux_arm](https://s3.amazonaws.com/benchmarks.redislabs/tools/redis-benchmark-go/redis-benchmark-go_linux_arm)    |
| Darwin   | amd64  (64-bit X86)     | [redis-benchmark-go_darwin_amd64](https://s3.amazonaws.com/benchmarks.redislabs/tools/redis-benchmark-go/redis-benchmark-go_darwin_amd64)    |
| Darwin   | arm64 (64-bit ARM)     | [redis-benchmark-go_darwin_arm64](https://s3.amazonaws.com/benchmarks.redislabs/tools/redis-benchmark-go/redis-benchmark-go_darwin_arm64)    |



Here's an example on how to use the above links:
```bash
# Fetch this repo
wget https://s3.amazonaws.com/benchmarks.redislabs/tools/redis-benchmark-go/redis-benchmark-go_linux_amd64

# change permissions
chmod 755 redis-benchmark-go_linux_amd64

# give it a try 
./redis-benchmark-go_linux_amd64 --help
```

## Installation in a Golang env

The easiest way to get and install the benchmark utility with a Go Env is to use
`go get` and then `go install`:
```bash
# Fetch this repo
go get github.com/filipecosta90/redis-benchmark-go
cd $GOPATH/src/github.com/filipecosta90/redis-benchmark-go
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
  -l    Loop. Run the tests forever.
  -n uint
        Total number of requests (default 100000)
  -p int
        Server port. (default 12000)
  -r uint
        keyspace length. The benchmark will expand the string __key__ inside an argument with a number in the specified range from 0 to keyspacelen-1. The substitution changes every time a command is executed. (default 1000000)
  -random-seed int
        random seed to be used. (default 12345)
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