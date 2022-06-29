package main

import (
	"encoding/json"
	"flag"
	"fmt"
	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
	"github.com/mediocregopher/radix/v3"
	"golang.org/x/time/rate"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
)

var totalCommands uint64
var totalErrors uint64
var latencies *hdrhistogram.Histogram
var instantLatencies *hdrhistogram.Histogram

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

func ingestionRoutine(conn radix.Client, enableMultiExec bool, datapointsChan chan datapoint, continueOnError bool, cmdS []string, keyspacelen, datasize, number_samples uint64, loop bool, debug_level int, wg *sync.WaitGroup, keyplace, dataplace, fieldplace int, fieldrange uint64, useLimiter bool, rateLimiter *rate.Limiter, waitReplicas, waitReplicasMs int) {
	defer wg.Done()
	for i := 0; uint64(i) < number_samples || loop; i++ {
		rawCurrentCmd, key, _ := keyBuildLogic(keyplace, dataplace, fieldplace, datasize, keyspacelen, fieldrange, cmdS)
		sendCmdLogic(conn, rawCurrentCmd, enableMultiExec, key, datapointsChan, continueOnError, debug_level, useLimiter, rateLimiter, waitReplicas, waitReplicasMs)
	}
}

func keyBuildLogic(keyPos int, dataPos int, fieldplace int, datasize, keyspacelen uint64, fieldrange uint64, cmdS []string) (cmd radix.CmdAction, key string, keySlot uint16) {
	newCmdS := cmdS
	if keyPos > -1 {
		newCmdS[keyPos] = fmt.Sprintf("%d", rand.Int63n(int64(keyspacelen)))
	}
	if fieldplace > -1 {
		newCmdS[fieldplace] = fmt.Sprintf("field-%d", rand.Int63n(int64(fieldrange)))
	}
	if dataPos > -1 {
		newCmdS[dataPos] = stringWithCharset(int(datasize), charset)
	}
	rawCmd := radix.Cmd(nil, newCmdS[0], newCmdS[1:]...)

	return rawCmd, key, radix.ClusterSlot([]byte(newCmdS[1]))
}

func sendCmdLogic(conn radix.Client, cmd radix.CmdAction, enableMultiExec bool, key string, datapointsChan chan datapoint, continueOnError bool, debug_level int, useRateLimiter bool, rateLimiter *rate.Limiter, waitReplicas, waitReplicasMs int) {
	if useRateLimiter {
		r := rateLimiter.ReserveN(time.Now(), int(1))
		time.Sleep(r.Delay())
	}
	var err error
	startT := time.Now()
	if enableMultiExec {
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
	} else if waitReplicas > 0 {
		// pipeline the command + wait
		err = conn.Do(radix.Pipeline(cmd,
			radix.Cmd(nil, "WAIT", fmt.Sprintf("%d", waitReplicas), fmt.Sprintf("%d", waitReplicasMs))))
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
	port := flag.Int("p", 6379, "Server port.")
	rps := flag.String("rps", "0", "Max rps. If 0 no limit is applied and the DB is stressed up to maximum. You can also provide a comma separated <timesecs>=<rps>,...")
	password := flag.String("a", "", "Password for Redis Auth.")
	jsonOutFile := flag.String("json-out-file", "", "Name of json output file, if not set, will not print to json.")
	seed := flag.Int64("random-seed", 12345, "random seed to be used.")
	clients := flag.String("c", "50", "number of clients. You can also provide a comma separated <timesecs>=<clients>,...")
	keyspacelen := flag.Uint64("r", 1000000, "keyspace length. The benchmark will expand the string __key__ inside an argument with a number in the specified range from 0 to keyspacelen-1. The substitution changes every time a command is executed.")
	fieldrange := flag.Uint64("field-range", 10, "field range length. The benchmark will expand the string __field__ inside an argument with a number in the specified range from 0 to fieldrange-1. The substitution changes every time a command is executed.")
	datasize := flag.Uint64("d", 3, "Data size of the expanded string __data__ value in bytes. The benchmark will expand the string __data__ inside an argument with a charset with length specified by this parameter. The substitution changes every time a command is executed.")
	numberRequests := flag.Uint64("n", 10000000, "Total number of requests")
	debug := flag.Int("debug", 0, "Client debug level.")
	multi := flag.Bool("multi", false, "Run each command in multi-exec.")
	waitReplicas := flag.Int("wait-replicas", 0, "If larger than 0 will wait for the specified number of replicas.")
	waitReplicasMs := flag.Int("wait-replicas-timeout-ms", 1000, "WAIT timeout when used together with -wait-replicas.")
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

	keyPlaceOlderPos := -1
	fieldPlaceOlderPos := -1
	dataPlaceOlderPos := -1
	for pos, arg := range args {
		if arg == "__data__" {
			dataPlaceOlderPos = pos
		}
		if arg == "__key__" {
			keyPlaceOlderPos = pos
		}
		if arg == "__field__" {
			fieldPlaceOlderPos = pos
		}
	}
	clientsArr := strings.Split(*clients, ",")

	seconds, clientsStart := extract(clientsArr[0])
	fmt.Println(seconds, clientsStart)
	for _, s := range clientsArr[1:] {
		seconds, clientsAtSec := extract(s)
		fmt.Println(seconds, clientsAtSec)
	}
	samplesPerClient := (*numberRequests) / uint64(clientsStart)
	client_update_tick := 1
	latencies = hdrhistogram.New(1, 90000000, 3)
	instantLatencies = hdrhistogram.New(1, 90000000, 3)
	opts := make([]radix.DialOpt, 0)
	if *password != "" {
		opts = append(opts, radix.DialAuthPass(*password))
	}
	ips, _ := net.LookupIP(*host)
	fmt.Printf("IPs %v\n", ips)

	stopChan := make(chan struct{})
	// a WaitGroup for the goroutines to tell us they've stopped
	wg := sync.WaitGroup{}
	if !*loop {
		fmt.Printf("Starting with a total of %d clients. Commands per client: %d Total commands: %d\n", clientsStart, samplesPerClient, *numberRequests)
	} else {
		fmt.Printf("Running in loop until you hit Ctrl+C\n")
	}
	fmt.Printf("Using random seed: %d\n", *seed)
	rand.Seed(*seed)
	var cluster *radix.Cluster

	var requestRate = Inf
	var requestBurst = clientsStart
	useRateLimiter := false
	var rateLimiter = rate.NewLimiter(requestRate, int(requestBurst))
	now := time.Now()
	if *rps != "0" {
		useRateLimiter = true
		rpsArr := strings.Split(*rps, ",")
		go applyRateLimit(rpsArr, now, rateLimiter)
	}

	datapointsChan := make(chan datapoint, *numberRequests)
	for channel_id := 1; channel_id <= clientsStart; channel_id++ {
		spinClient(wg, ips, port, clusterMode, cluster, opts, clientsStart, debug, channel_id, args, multi, datapointsChan, keyspacelen, datasize, samplesPerClient, loop, keyPlaceOlderPos, dataPlaceOlderPos, fieldPlaceOlderPos, fieldrange, useRateLimiter, rateLimiter, waitReplicas, waitReplicasMs)
		// delay the creation 10ms for each additional client
		time.Sleep(time.Millisecond * 10)
	}

	// listen for C-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	tick := time.NewTicker(time.Duration(client_update_tick) * time.Second)
	closed, start, duration, totalMessages, commandRateTs, p50LatenciesTs := updateCLI(tick, c, *numberRequests, *loop, datapointsChan)
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

	if strings.Compare(*jsonOutFile, "") != 0 {

		res := testResult{
			StartTime:      start.Unix(),
			Duration:       duration.Seconds(),
			CommandRate:    messageRate,
			TotalCommands:  totalMessages,
			CommandRateTs:  commandRateTs,
			P50LatenciesTs: p50LatenciesTs,
		}
		file, err := json.MarshalIndent(res, "", " ")
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile(*jsonOutFile, file, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	if closed {
		return
	}

	// tell the goroutine to stop
	close(stopChan)
	// and wait for them both to reply back
	wg.Wait()
}

func extract(clientsArr string) (int, int) {
	clientsDetail := strings.Split(clientsArr, "=")
	seconds := 0
	var clientsStart int
	if len(clientsDetail) == 1 {
		clientsStart, _ = strconv.Atoi(clientsDetail[0])
	} else {
		seconds, _ = strconv.Atoi(clientsDetail[0])
		clientsStart, _ = strconv.Atoi(clientsDetail[1])
	}
	return seconds, clientsStart
}

func spinClient(wg sync.WaitGroup, ips []net.IP, port *int, clusterMode *bool, cluster *radix.Cluster, opts []radix.DialOpt, clientsStart int, debug *int, channel_id int, args []string, multi *bool, datapointsChan chan datapoint, keyspacelen *uint64, datasize *uint64, samplesPerClient uint64, loop *bool, keyPlaceOlderPos int, dataPlaceOlderPos int, fieldPlaceOlderPos int, fieldlen *uint64, useRateLimiter bool, rateLimiter *rate.Limiter, waitReplicas *int, waitReplicasMs *int) {
	wg.Add(1)
	connectionStr := fmt.Sprintf("%s:%d", ips[rand.Int63n(int64(len(ips)))], *port)
	if *clusterMode {
		cluster = getOSSClusterConn(connectionStr, opts, clientsStart)
	}
	if *debug > 0 {
		fmt.Printf("Using connection string %s for client %d\n", connectionStr, channel_id)
	}
	cmd := make([]string, len(args))
	copy(cmd, args)
	if *clusterMode {
		go ingestionRoutine(cluster, *multi, datapointsChan, true, cmd, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, keyPlaceOlderPos, dataPlaceOlderPos, fieldPlaceOlderPos, *fieldlen, useRateLimiter, rateLimiter, *waitReplicas, *waitReplicasMs)
	} else {
		go ingestionRoutine(getStandaloneConn(connectionStr, opts, 1), *multi, datapointsChan, true, cmd, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, keyPlaceOlderPos, dataPlaceOlderPos, fieldPlaceOlderPos, *fieldlen, useRateLimiter, rateLimiter, *waitReplicas, *waitReplicasMs)
	}
}

func applyRateLimit(rpsArr []string, now time.Time, rateLimiter *rate.Limiter) {
	rpsDetail := strings.Split(rpsArr[0], "=")
	seconds := 0
	previousSecs := seconds
	newLimit := 0
	if len(rpsDetail) == 1 {
		newLimit, _ = strconv.Atoi(rpsDetail[0])
	} else {
		seconds, _ = strconv.Atoi(rpsDetail[0])
		newLimit, _ = strconv.Atoi(rpsDetail[1])
	}
	duration := time.Duration(seconds)
	newTime := now.Add(duration)
	fmt.Fprintf(os.Stdout, "Setting an initial target rate of %d rps\n", newLimit)
	rateLimiter.SetLimitAt(newTime, rate.Limit(newLimit))
	for _, rpsPair := range rpsArr[1:] {
		rpsDetail = strings.Split(rpsPair, "=")
		seconds, _ = strconv.Atoi(rpsDetail[0])
		duration = time.Duration(seconds)
		newLimit, _ = strconv.Atoi(rpsDetail[1])
		time.Sleep((time.Duration(seconds-previousSecs) * time.Second))
		newTime = time.Now()
		previousSecs = seconds
		fmt.Fprintf(os.Stdout, "\nSetting a new target %d rps at time %v (after %d secs)\n", newLimit, newTime, duration)
		rateLimiter.SetLimitAt(newTime, rate.Limit(newLimit))
	}
}

func updateCLI(tick *time.Ticker, c chan os.Signal, message_limit uint64, loop bool, datapointsChan chan datapoint) (bool, time.Time, time.Duration, uint64, []float64, []float64) {
	var currentErr uint64 = 0
	var currentCount uint64 = 0
	start := time.Now()
	prevTime := time.Now()
	prevMessageCount := uint64(0)
	messageRateTs := []float64{}
	p50LatenciesTs := []float64{}
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
				p50LatenciesTs = append(p50LatenciesTs, p50)

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
					return true, start, time.Since(start), totalCommands, messageRateTs, p50LatenciesTs
				}
				break
			}

		case <-c:
			fmt.Println("\nreceived Ctrl-c - shutting down")
			return true, start, time.Since(start), totalCommands, messageRateTs, p50LatenciesTs
		}
	}
}
