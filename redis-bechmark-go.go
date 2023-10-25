package main

import (
	"context"
	"flag"
	"fmt"
	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
	radix "github.com/mediocregopher/radix/v4"
	"github.com/redis/rueidis"
	"golang.org/x/time/rate"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

func benchmarkRoutine(radixClient Client, ruedisClient rueidis.Client, useRuedis, useCSC, enableMultiExec bool, datapointsChan chan datapoint, continueOnError bool, cmdS [][]string, commandsCDF []float32, keyspacelen, datasize, number_samples uint64, loop bool, debug_level int, wg *sync.WaitGroup, keyplace, dataplace []int, readOnly []bool, useLimiter bool, rateLimiter *rate.Limiter, waitReplicas, waitReplicasMs int, cscDuration time.Duration) {
	defer wg.Done()
	for i := 0; uint64(i) < number_samples || loop; i++ {
		cmdPos := sample(commandsCDF)
		kplace := keyplace[cmdPos]
		dplace := dataplace[cmdPos]
		isReadOnly := readOnly[cmdPos]
		cmds := cmdS[cmdPos]
		newCmdS, key := keyBuildLogic(kplace, dplace, datasize, keyspacelen, cmds, charset)
		if useLimiter {
			r := rateLimiter.ReserveN(time.Now(), int(1))
			time.Sleep(r.Delay())
		}
		if useRuedis {
			sendCmdLogicRuedis(ruedisClient, newCmdS, enableMultiExec, datapointsChan, continueOnError, debug_level, useCSC, isReadOnly, cscDuration, waitReplicas, waitReplicasMs)
		} else {
			sendCmdLogicRadix(radixClient, newCmdS, enableMultiExec, key, datapointsChan, continueOnError, debug_level, waitReplicas, waitReplicasMs)

		}
	}
}

func sendCmdLogicRuedis(ruedisClient rueidis.Client, newCmdS []string, enableMultiExec bool, datapointsChan chan datapoint, continueOnError bool, debug_level int, useCSC, isReadOnly bool, cscDuration time.Duration, waitReplicas, waitReplicasMs int) {
	ctx := context.Background()
	var startT time.Time
	var endT time.Time
	var redisResult rueidis.RedisResult
	cacheHit := false
	var err error
	arbitrary := ruedisClient.B().Arbitrary(newCmdS[0])
	if len(newCmdS) > 1 {
		arbitrary = arbitrary.Keys(newCmdS[1])
		if len(newCmdS) > 2 {
			arbitrary = arbitrary.Args(newCmdS[2:]...)
		}
	}
	if useCSC && isReadOnly {
		startT = time.Now()
		redisResult = ruedisClient.DoCache(ctx, arbitrary.Cache(), cscDuration)
		endT = time.Now()
	} else if enableMultiExec {
		cmds := make(rueidis.Commands, 0, 3)
		cmds = append(cmds, ruedisClient.B().Multi().Build())
		cmds = append(cmds, arbitrary.Build())
		cmds = append(cmds, ruedisClient.B().Exec().Build())
		startT = time.Now()
		resMulti := ruedisClient.DoMulti(ctx, cmds...)
		endT = time.Now()
		redisResult = resMulti[1]
	} else if waitReplicas > 0 {
		cmds := make(rueidis.Commands, 0, 2)
		cmds = append(cmds, arbitrary.Build())
		cmds = append(cmds, ruedisClient.B().Wait().Numreplicas(int64(waitReplicas)).Timeout(int64(waitReplicasMs)).Build())
		startT = time.Now()
		resMulti := ruedisClient.DoMulti(ctx, cmds...)
		endT = time.Now()
		redisResult = resMulti[0]
	} else {
		startT = time.Now()
		redisResult = ruedisClient.Do(ctx, arbitrary.Build())
		endT = time.Now()
	}
	err = redisResult.NonRedisError()
	cacheHit = redisResult.IsCacheHit()

	if err != nil {
		if continueOnError {
			if debug_level > 0 {
				log.Println(fmt.Sprintf("Received an error with the following command(s): %v, error: %v", newCmdS, err))
			}
		} else {
			log.Fatalf("Received an error with the following command(s): %v, error: %v", newCmdS, err)
		}
	}
	duration := endT.Sub(startT)
	datapointsChan <- datapoint{!(err != nil), duration.Microseconds(), cacheHit}
}

func sendCmdLogicRadix(conn Client, newCmdS []string, enableMultiExec bool, key string, datapointsChan chan datapoint, continueOnError bool, debug_level int, waitReplicas, waitReplicasMs int) {
	cmd := radix.Cmd(nil, newCmdS[0], newCmdS[1:]...)
	ctx := context.Background()
	cacheHit := false
	var err error
	startT := time.Now()
	if enableMultiExec {
		err = conn.Do(ctx, radix.WithConn(key, func(ctx context.Context, c radix.Conn) error {

			// Begin the transaction with a MULTI command
			if err := conn.Do(ctx, radix.Cmd(nil, "MULTI")); err != nil {
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
					conn.Do(ctx, radix.Cmd(nil, "DISCARD"))
					log.Fatalf("Received an error while in multi: %v, error: %v", cmd, err)
				}
			}()

			// queue up the transaction's commands
			err = conn.Do(ctx, cmd)

			// execute the transaction, capturing the result in a Tuple. We only
			// care about the first element (the result from GET), so we discard the
			// second by setting nil.
			return conn.Do(ctx, radix.Cmd(nil, "EXEC"))
		}))
	} else if waitReplicas > 0 {
		// Create a new pipeline for the WAIT command
		p := radix.NewPipeline()
		// Pass both cmd and waitCmd to the original pipeline
		p.Append(cmd)
		p.Append(radix.Cmd(nil, "WAIT", fmt.Sprintf("%d", waitReplicas), fmt.Sprintf("%d", waitReplicasMs)))
		err = conn.Do(ctx, p)
	} else {
		err = conn.Do(ctx, cmd)
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
	datapointsChan <- datapoint{!(err != nil), duration.Microseconds(), cacheHit}
}

func main() {
	host := flag.String("h", "127.0.0.1", "Server hostname.")
	port := flag.Int("p", 12000, "Server port.")
	rps := flag.Int64("rps", 0, "Max rps. If 0 no limit is applied and the DB is stressed up to maximum.")
	rpsburst := flag.Int64("rps-burst", 0, "Max rps burst. If 0 the allowed burst will be the ammount of clients.")
	username := flag.String("u", "", "Username for Redis Auth.")
	password := flag.String("a", "", "Password for Redis Auth.")
	seed := flag.Int64("random-seed", 12345, "random seed to be used.")
	clients := flag.Uint64("c", 50, "number of clients.")
	keyspacelen := flag.Uint64("r", 1000000, "keyspace length. The benchmark will expand the string __key__ inside an argument with a number in the specified range from 0 to keyspacelen-1. The substitution changes every time a command is executed.")
	datasize := flag.Uint64("d", 3, "Data size of the expanded string __data__ value in bytes. The benchmark will expand the string __data__ inside an argument with a charset with length specified by this parameter. The substitution changes every time a command is executed.")
	numberRequests := flag.Uint64("n", 10000000, "Total number of requests")
	debug := flag.Int("debug", 0, "Client debug level.")
	multi := flag.Bool("multi", false, "Run each command in multi-exec.")
	waitReplicas := flag.Int("wait-replicas", 0, "If larger than 0 will wait for the specified number of replicas.")
	waitReplicasMs := flag.Int("wait-replicas-timeout-ms", 1000, "WAIT timeout when used together with -wait-replicas.")
	clusterMode := flag.Bool("oss-cluster", false, "Enable OSS cluster mode.")
	loop := flag.Bool("l", false, "Loop. Run the tests forever.")
	betweenClientsDelay := flag.Duration("between-clients-duration", time.Millisecond*0, "Between each client creation, wait this time.")
	version := flag.Bool("v", false, "Output version and exit")
	verbose := flag.Bool("verbose", false, "Output verbose info")
	cscEnabled := flag.Bool("csc", false, "Enable client side caching")
	useRuedis := flag.Bool("rueidis", false, "Use rueidis as the vanilla underlying client.")
	cscDuration := flag.Duration("csc-ttl", time.Minute, "Client side cache ttl for cached entries")
	clientKeepAlive := flag.Duration("client-keepalive", time.Minute, "Client keepalive")
	cscSizeBytes := flag.Int("csc-per-client-bytes", rueidis.DefaultCacheBytes, "client side cache size that bind to each TCP connection to a single redis instance")
	continueonerror := flag.Bool("continue-on-error", false, "Output verbose info")
	resp := flag.String("resp", "", "redis command response protocol (2 - RESP 2, 3 - RESP 3). If empty will not enforce it.")
	nameserver := flag.String("nameserver", "", "the IP address of the DNS name server. The IP address can be an IPv4 or an IPv6 address. If empty will use the default host namserver.")
	flag.Var(&benchmarkCommands, "cmd", "Specify a query to send in quotes. Each command that you specify is run with its ratio. For example:-cmd=\"SET __key__ __value__\" -cmd-ratio=1")
	flag.Var(&benchmarkCommandsRatios, "cmd-ratio", "The query ratio vs other queries used in the same benchmark. Each command that you specify is run with its ratio. For example: -cmd=\"SET __key__ __value__\" -cmd-ratio=0.8 -cmd=\"GET __key__\"  -cmd-ratio=0.2")

	flag.Parse()
	totalQueries := len(benchmarkCommands)
	if totalQueries == 0 {
		totalQueries = 1
		benchmarkCommands = make([]string, totalQueries)
	}
	cmds := make([][]string, totalQueries)
	cmdRates := make([]float64, totalQueries)
	cmdKeyplaceHolderPos := make([]int, totalQueries)
	cmdDataplaceHolderPos := make([]int, totalQueries)
	cmdReadOnly := make([]bool, totalQueries)
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
	cdf := []float32{1.0}
	if len(args) > 0 {
		if len(args) < 2 {
			log.Fatalf("You need to specify a command after the flag command arguments. The commands requires a minimum size of 2 ( command name and key )")
		}
		cmds[0] = args

	} else {
		if *verbose {
			fmt.Println("checking for -cmd args")
		}
		_, cdf = prepareCommandsDistribution(benchmarkCommands, cmds, cmdRates)
	}

	for i := 0; i < len(cmds); i++ {
		cmdKeyplaceHolderPos[i], cmdDataplaceHolderPos[i] = getplaceholderpos(cmds[i], *verbose)
		cmdAllCaps := strings.ToUpper(cmds[i][0])
		_, isReadOnly := readOnlyCommands[cmdAllCaps]
		cmdReadOnly[i] = isReadOnly
	}

	var requestRate = Inf
	var requestBurst = int(*rps)
	useRateLimiter := false
	if *rps != 0 {
		requestRate = rate.Limit(*rps)
		useRateLimiter = true
		if *rpsburst != 0 {
			requestBurst = int(*rpsburst)
		}
	}

	var rateLimiter = rate.NewLimiter(requestRate, requestBurst)

	samplesPerClient := *numberRequests / *clients
	client_update_tick := 1
	latencies = hdrhistogram.New(1, 90000000, 3)
	opts := radix.Dialer{}
	if *password != "" {
		opts.AuthPass = *password
	}
	if *username != "" {
		opts.AuthUser = *username
	}
	alwaysRESP2 := false
	if *resp == "2" {
		opts.Protocol = "2"
		alwaysRESP2 = true
	} else if *resp == "3" {
		opts.Protocol = "3"
		alwaysRESP2 = false
	}

	ips := make([]net.IP, 0)
	if *nameserver != "" {
		fmt.Printf("Using %s to resolve hostname %s\n", *nameserver, *host)
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * time.Duration(10000),
				}
				return d.DialContext(ctx, network, *nameserver)
			},
		}
		ips, _ = r.LookupIP(context.Background(), "ip", *host)
	} else {
		ips, _ = net.LookupIP(*host)
	}
	if len(ips) < 1 {
		log.Fatalf("Failed to resolve %s to any IP", *host)
	}

	fmt.Printf("IPs %v\n", ips)

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
	var radixStandalone radix.Client
	var ruedisClient rueidis.Client
	var err error = nil
	datapointsChan := make(chan datapoint, *numberRequests)
	for clientId := 1; uint64(clientId) <= *clients; clientId++ {
		wg.Add(1)
		connectionStr := fmt.Sprintf("%s:%d", ips[rand.Int63n(int64(len(ips)))], *port)
		if *verbose {
			fmt.Printf("Using connection string %s for client %d\n", connectionStr, clientId)
		}
		cmd := make([]string, len(args))
		copy(cmd, args)
		if *cscEnabled || *useRuedis {
			clientOptions := rueidis.ClientOption{
				InitAddress:         []string{connectionStr},
				Username:            *username,
				Password:            *password,
				AlwaysPipelining:    false,
				AlwaysRESP2:         alwaysRESP2,
				DisableCache:        !*cscEnabled,
				BlockingPoolSize:    0,
				PipelineMultiplex:   0,
				RingScaleEachConn:   1,
				ReadBufferEachConn:  1024,
				WriteBufferEachConn: 1024,
				CacheSizeEachConn:   *cscSizeBytes,
			}
			clientOptions.Dialer.KeepAlive = *clientKeepAlive
			ruedisClient, err = rueidis.NewClient(clientOptions)
			if err != nil {
				panic(err)
			}
			go benchmarkRoutine(radixStandalone, ruedisClient, *useRuedis, *cscEnabled, *multi, datapointsChan, *continueonerror, cmds, cdf, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, cmdKeyplaceHolderPos, cmdDataplaceHolderPos, cmdReadOnly, useRateLimiter, rateLimiter, *waitReplicas, *waitReplicasMs, *cscDuration)
		} else {
			// legacy radix code
			if *clusterMode {
				cluster = getOSSClusterConn(connectionStr, opts, 1)
				go benchmarkRoutine(cluster, ruedisClient, *useRuedis, *cscEnabled, *multi, datapointsChan, *continueonerror, cmds, cdf, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, cmdKeyplaceHolderPos, cmdDataplaceHolderPos, cmdReadOnly, useRateLimiter, rateLimiter, *waitReplicas, *waitReplicasMs, *cscDuration)
			} else {
				radixStandalone = getStandaloneConn(connectionStr, opts, 1)
				go benchmarkRoutine(radixStandalone, ruedisClient, *useRuedis, *cscEnabled, *multi, datapointsChan, *continueonerror, cmds, cdf, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, cmdKeyplaceHolderPos, cmdDataplaceHolderPos, cmdReadOnly, useRateLimiter, rateLimiter, *waitReplicas, *waitReplicasMs, *cscDuration)
			}
		}

		// delay the creation for each additional client
		time.Sleep(*betweenClientsDelay)
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

	os.Exit(0)
}

func updateCLI(tick *time.Ticker, c chan os.Signal, message_limit uint64, loop bool, datapointsChan chan datapoint) (bool, time.Time, time.Duration, uint64, []float64) {
	var currentErr uint64 = 0
	var currentCount uint64 = 0
	var currentCachedCount uint64 = 0
	start := time.Now()
	prevTime := time.Now()
	prevMessageCount := uint64(0)
	messageRateTs := []float64{}
	var dp datapoint
	fmt.Printf("%26s %7s %25s %25s %7s %25s %25s %7s %25s\n", "Test time", " ", "Total Commands", "Total Errors", "", "Command Rate", "Client Cache Hits", "", "p50 lat. (msec)")
	for {
		select {
		case dp = <-datapointsChan:
			{
				latencies.RecordValue(dp.duration_ms)
				if !dp.success {
					currentErr++
				}
				if dp.cachedEntry {
					currentCachedCount++
				}
				currentCount++
			}
		case <-tick.C:
			{
				totalCommands += currentCount
				totalCached += currentCachedCount
				totalErrors += currentErr
				currentErr = 0
				currentCount = 0
				currentCachedCount = 0
				now := time.Now()
				took := now.Sub(prevTime)
				messageRate := float64(totalCommands-prevMessageCount) / float64(took.Seconds())
				completionPercentStr := "[----%]"
				if !loop {
					completionPercent := float64(totalCommands) / float64(message_limit) * 100.0
					completionPercentStr = fmt.Sprintf("[%3.1f%%]", completionPercent)
				}
				errorPercent := float64(totalErrors) / float64(totalCommands) * 100.0
				cachedPercent := 0.0
				if totalCached > 0 {
					cachedPercent = float64(totalCached) / float64(totalCommands) * 100.0
				}

				p50 := float64(latencies.ValueAtQuantile(50.0)) / 1000.0

				if prevMessageCount == 0 && totalCommands != 0 {
					start = time.Now()
				}
				if totalCommands != 0 {
					messageRateTs = append(messageRateTs, messageRate)
				}
				prevMessageCount = totalCommands
				prevTime = now

				fmt.Printf("%25.0fs %s %25d %25d [%3.1f%%] %25.0f %25d [%3.1f%%] %25.3f\t", time.Since(start).Seconds(), completionPercentStr, totalCommands, totalErrors, errorPercent, messageRate, totalCached, cachedPercent, p50)
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
