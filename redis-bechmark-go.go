package main

import (
	"flag"
	"fmt"
	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
	"github.com/mediocregopher/radix/v3"
	"golang.org/x/time/rate"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

var totalCommands uint64
var totalErrors uint64
var latencies *hdrhistogram.Histogram

const Inf = rate.Limit(math.MaxFloat64)
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type datapoint struct {
	success     bool
	duration_ms int64
}

func stringWithCharset(length int, charset string) string {

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func ingestionRoutine(conn radix.Client, enableMultiExec bool, datapointsChan chan datapoint, continueOnError bool, cmdS []string, keyspacelen, datasize, number_samples uint64, loop bool, debug_level int, wg *sync.WaitGroup, keyplace, dataplace int, useLimiter bool, rateLimiter *rate.Limiter) {
	defer wg.Done()
	for i := 0; uint64(i) < number_samples || loop; i++ {
		rawCurrentCmd, _, _ := keyBuildLogic(keyplace, dataplace, datasize, keyspacelen, cmdS)
		sendCmdLogic(conn, rawCurrentCmd, enableMultiExec, datapointsChan, continueOnError, debug_level, useLimiter, rateLimiter)
	}
}

func keyBuildLogic(keyPos int, dataPos int, datasize, keyspacelen uint64, cmdS []string) (cmd radix.CmdAction, key string, keySlot uint16) {
	newCmdS := cmdS
	if keyPos > -1 {
		newCmdS[keyPos] = fmt.Sprintf("%d", rand.Int63n(int64(keyspacelen)))
	}
	if dataPos > -1 {
		newCmdS[dataPos] = stringWithCharset(int(datasize), charset)
	}
	rawCmd := radix.Cmd(nil, newCmdS[0], newCmdS[1:]...)

	return rawCmd, key, radix.ClusterSlot([]byte(newCmdS[1]))
}

func sendCmdLogic(conn radix.Client, cmd radix.CmdAction, enableMultiExec bool, datapointsChan chan datapoint, continueOnError bool, debug_level int, useRateLimiter bool, rateLimiter *rate.Limiter) {
	if useRateLimiter {
		r := rateLimiter.ReserveN(time.Now(), int(1))
		time.Sleep(r.Delay())
	}
	var err error
	startT := time.Now()
	if enableMultiExec {
		key := "userFriends"
		err = conn.Do(radix.WithConn(key, func(c radix.Conn) error {

			// Begin the transaction with a MULTI command
			if err := conn.Do(radix.Cmd(nil, "MULTI")); err != nil {
				log.Fatalf("Received an error while preparing for MULTI: %v, error: %v", cmd, err)
			}

			// If any of the calls after the MULTI call error it's important that
			// the transaction is discarded. This isn't strictly necessary if the
			// only possible error is a network error, as the connection would be
			// closed by the client anyway.
			var err error
			defer func() {
				if err != nil {
					// The return from DISCARD doesn't matter. If it's an error then
					// it's a network error and the Conn will be closed by the
					// client.
					conn.Do(radix.Cmd(nil, "DISCARD"))
					log.Fatalf("Received an error while in multi: %v, error: %v", cmd, err)
				}
			}()

			// queue up the transaction's commands
			err = conn.Do(cmd)

			// execute the transaction, capturing the result in a Tuple. We only
			// care about the first element (the result from GET), so we discard the
			// second by setting nil.
			return conn.Do(radix.Cmd(nil, "EXEC"))
		}))
	} else {
		err = conn.Do(cmd)
	}
	endT := time.Now()
	if err != nil {
		if continueOnError {
			if debug_level > 0 {
				log.Println(fmt.Sprintf("Received an error with the following command(s): %v, error: %v", cmd, err))
			}
		} else {
			log.Fatalf("Received an error with the following command(s): %v, error: %v", cmd, err)
		}
	}
	duration := endT.Sub(startT)
	datapointsChan <- datapoint{!(err != nil), duration.Microseconds()}
}

func main() {
	host := flag.String("h", "127.0.0.1", "Server hostname.")
	port := flag.Int("p", 12000, "Server port.")
	rps := flag.Int64("rps", 0, "Max rps. If 0 no limit is applied and the DB is stressed up to maximum.")
	password := flag.String("a", "", "Password for Redis Auth.")
	seed := flag.Int64("random-seed", 12345, "random seed to be used.")
	clients := flag.Uint64("c", 50, "number of clients.")
	keyspacelen := flag.Uint64("r", 1000000, "keyspace length. The benchmark will expand the string __key__ inside an argument with a number in the specified range from 0 to keyspacelen-1. The substitution changes every time a command is executed.")
	datasize := flag.Uint64("d", 3, "Data size of the expanded string __data__ value in bytes. The benchmark will expand the string __data__ inside an argument with a charset with length specified by this parameter. The substitution changes every time a command is executed.")
	numberRequests := flag.Uint64("n", 10000000, "Total number of requests")
	debug := flag.Int("debug", 0, "Client debug level.")
	multi := flag.Bool("multi", false, "Run each command in multi-exec.")
	clusterMode := flag.Bool("oss-cluster", false, "Enable OSS cluster mode.")
	loop := flag.Bool("l", false, "Loop. Run the tests forever.")
	version := flag.Bool("v", false, "Output version and exit")
	flag.Parse()
	git_sha := toolGitSHA1()
	git_dirty_str := ""
	if toolGitDirty() {
		git_dirty_str = "-dirty"
	}
	if *version {
		fmt.Fprintf(os.Stdout, "redis-benchmark-go (git_sha1:%s%s)\n", git_sha, git_dirty_str)
		os.Exit(0)
	}
	args := flag.Args()
	if len(args) < 2 {
		log.Fatalf("You need to specify a command after the flag command arguments. The commands requires a minimum size of 2 ( command name and key )")
	}

	var requestRate = Inf
	var requestBurst = 1
	useRateLimiter := false
	if *rps != 0 {
		requestRate = rate.Limit(*rps)
		requestBurst = int(*clients)
		useRateLimiter = true
	}

	var rateLimiter = rate.NewLimiter(requestRate, requestBurst)

	keyPlaceOlderPos := -1
	dataPlaceOlderPos := -1
	for pos, arg := range args {
		if arg == "__data__" {
			dataPlaceOlderPos = pos
		}
		if arg == "__key__" {
			keyPlaceOlderPos = pos
		}
	}
	samplesPerClient := *numberRequests / *clients
	client_update_tick := 1
	latencies = hdrhistogram.New(1, 90000000, 3)
	opts := make([]radix.DialOpt, 0)
	if *password != "" {
		opts = append(opts, radix.DialAuthPass(*password))
	}
	ips, _ := net.LookupIP(*host)

	stopChan := make(chan struct{})
	// a WaitGroup for the goroutines to tell us they've stopped
	wg := sync.WaitGroup{}
	if !*loop {
		fmt.Printf("Total clients: %d. Commands per client: %d Total commands: %d\n", *clients, samplesPerClient, *numberRequests)
	} else {
		fmt.Printf("Running in loop until you hit Ctrl+C\n")
	}
	fmt.Printf("Using random seed: %d\n", *seed)
	rand.Seed(*seed)
	var cluster *radix.Cluster

	datapointsChan := make(chan datapoint, *numberRequests)
	for channel_id := 1; uint64(channel_id) <= *clients; channel_id++ {
		wg.Add(1)
		connectionStr := fmt.Sprintf("%s:%d", ips[rand.Int63n(int64(len(ips)))], *port)
		if *clusterMode {
			cluster = getOSSClusterConn(connectionStr, opts, *clients)
		}
		fmt.Printf("Using connection string %s for client %d\n", connectionStr, channel_id)
		cmd := make([]string, len(args))
		copy(cmd, args)
		if *clusterMode {
			go ingestionRoutine(cluster, *multi, datapointsChan, true, cmd, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, keyPlaceOlderPos, dataPlaceOlderPos, useRateLimiter, rateLimiter)
		} else {
			if *multi {
				go ingestionRoutine(getStandaloneConn(connectionStr, opts, 1), *multi, datapointsChan, true, cmd, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, keyPlaceOlderPos, dataPlaceOlderPos, useRateLimiter, rateLimiter)
			} else {
				go ingestionRoutine(getStandaloneConn(connectionStr, opts, 1), *multi, datapointsChan, true, cmd, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, keyPlaceOlderPos, dataPlaceOlderPos, useRateLimiter, rateLimiter)
				time.Sleep(time.Millisecond * 10)
			}
		}
	}

	// listen for C-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	tick := time.NewTicker(time.Duration(client_update_tick) * time.Second)
	closed, _, duration, totalMessages, _ := updateCLI(tick, c, *numberRequests, *loop, datapointsChan)
	messageRate := float64(totalMessages) / float64(duration.Seconds())
	avgMs := float64(latencies.Mean()) / 1000.0
	p50IngestionMs := float64(latencies.ValueAtQuantile(50.0)) / 1000.0
	p95IngestionMs := float64(latencies.ValueAtQuantile(95.0)) / 1000.0
	p99IngestionMs := float64(latencies.ValueAtQuantile(99.0)) / 1000.0

	fmt.Printf("\n")
	fmt.Printf("#################################################\n")
	fmt.Printf("Total Duration %.3f Seconds\n", duration.Seconds())
	fmt.Printf("Total Errors %d\n", totalErrors)
	fmt.Printf("Throughput summary: %.0f requests per second\n", messageRate)
	fmt.Printf("Latency summary (msec):\n")
	fmt.Printf("    %9s %9s %9s %9s\n", "avg", "p50", "p95", "p99")
	fmt.Printf("    %9.3f %9.3f %9.3f %9.3f\n", avgMs, p50IngestionMs, p95IngestionMs, p99IngestionMs)

	if closed {
		return
	}

	// tell the goroutine to stop
	close(stopChan)
	// and wait for them both to reply back
	wg.Wait()
}

func updateCLI(tick *time.Ticker, c chan os.Signal, message_limit uint64, loop bool, datapointsChan chan datapoint) (bool, time.Time, time.Duration, uint64, []float64) {
	var currentErr uint64 = 0
	var currentCount uint64 = 0
	start := time.Now()
	prevTime := time.Now()
	prevMessageCount := uint64(0)
	messageRateTs := []float64{}
	var dp datapoint
	fmt.Printf("%26s %7s %25s %25s %7s %25s %25s\n", "Test time", " ", "Total Commands", "Total Errors", "", "Command Rate", "p50 lat. (msec)")
	for {
		select {
		case dp = <-datapointsChan:
			{
				latencies.RecordValue(dp.duration_ms)
				if !dp.success {
					currentErr++
				}
				currentCount++
			}
		case <-tick.C:
			{
				totalCommands += currentCount
				totalErrors += currentErr
				currentErr = 0
				currentCount = 0
				now := time.Now()
				took := now.Sub(prevTime)
				messageRate := float64(totalCommands-prevMessageCount) / float64(took.Seconds())
				completionPercentStr := "[----%]"
				if !loop {
					completionPercent := float64(totalCommands) / float64(message_limit) * 100.0
					completionPercentStr = fmt.Sprintf("[%3.1f%%]", completionPercent)
				}
				errorPercent := float64(totalErrors) / float64(totalCommands) * 100.0

				p50 := float64(latencies.ValueAtQuantile(50.0)) / 1000.0

				if prevMessageCount == 0 && totalCommands != 0 {
					start = time.Now()
				}
				if totalCommands != 0 {
					messageRateTs = append(messageRateTs, messageRate)
				}
				prevMessageCount = totalCommands
				prevTime = now

				fmt.Printf("%25.0fs %s %25d %25d [%3.1f%%] %25.2f %25.2f\t", time.Since(start).Seconds(), completionPercentStr, totalCommands, totalErrors, errorPercent, messageRate, p50)
				fmt.Printf("\r")
				//w.Flush()
				if message_limit > 0 && totalCommands >= uint64(message_limit) && !loop {
					return true, start, time.Since(start), totalCommands, messageRateTs
				}

				break
			}

		case <-c:
			fmt.Println("\nreceived Ctrl-c - shutting down")
			return true, start, time.Since(start), totalCommands, messageRateTs
		}
	}
}
