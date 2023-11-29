module github.com/filipecosta90/redis-benchmark-go

go 1.20

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/mediocregopher/radix/v4 v4.1.2
	github.com/redis/rueidis v1.0.19
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
)

require (
	github.com/tilinna/clock v1.0.2 // indirect
	golang.org/x/exp v0.0.0-20200224162631-6cc2880d07d6 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/redis/rueidis => github.com/filipecosta90/rueidis v0.0.0-20231129020706-4fbfa4b6c663
