package main

import (
	"flag"
	"fmt"
	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
	"github.com/mediocregopher/radix/v3"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

var totalCommands uint64
var totalErrors uint64
var latencies *hdrhistogram.Histogram

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func stringWithCharset(length int, charset string) string {

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func ingestionRoutine(cluster *radix.Cluster, continueOnError bool, cmdS []string, keyspacelen, datasize, number_samples uint64, loop bool, debug_level int, wg *sync.WaitGroup, keyplace, dataplace int) {
	defer wg.Done()
	for i := 0; uint64(i) < number_samples || loop; i++ {
		rawCurrentCmd, _, _ := keyBuildLogic(keyplace, dataplace, datasize, keyspacelen, cmdS)
		sendCmdLogic(cluster, rawCurrentCmd, continueOnError, debug_level)
	}
}

func getOSSClusterConn(addr string, opts []radix.DialOpt, clients uint64) *radix.Cluster {
	var vanillaCluster *radix.Cluster
	var err error

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr, opts...,
		)
	}

	// this cluster will use the ClientFunc to create a pool to each node in the
	// cluster.
	poolFunc := func(network, addr string) (radix.Client, error) {
		return radix.NewPool(network, addr, int(clients), radix.PoolConnFunc(customConnFunc), radix.PoolPipelineWindow(0, 0))
	}

	vanillaCluster, err = radix.NewCluster([]string{addr}, radix.ClusterPoolFunc(poolFunc))
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while creating new connection. error = %v", err)
	}
	// Issue CLUSTER SLOTS command
	err = vanillaCluster.Sync()
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while issuing CLUSTER SLOTS. error = %v", err)
	}
	return vanillaCluster
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

func sendCmdLogic(conn radix.Client, cmd radix.CmdAction, continueOnError bool, debug_level int) {
	startT := time.Now()
	var err = conn.Do(cmd)
	endT := time.Now()
	if err != nil {
		if continueOnError {
			atomic.AddUint64(&totalErrors, uint64(1))
			if debug_level > 0 {
				log.Println(fmt.Sprintf("Received an error with the following command(s): %v, error: %v", cmd, err))
			}
		} else {
			log.Fatalf("Received an error with the following command(s): %v, error: %v", cmd, err)
		}
	}
	duration := endT.Sub(startT)
	err = latencies.RecordValue(duration.Microseconds())
	if err != nil {
		log.Fatalf("Received an error while recording latencies: %v", err)
	}
	atomic.AddUint64(&totalCommands, uint64(1))
}

func main() {
	host := flag.String("h", "127.0.0.1", "Server hostname.")
	port := flag.Int("p", 12000, "Server port.")
	password := flag.String("a", "", "Password for Redis Auth.")
	seed := flag.Int64("random-seed", 12345, "random seed to be used.")
	clients := flag.Uint64("c", 50, "number of clients.")
	keyspacelen := flag.Uint64("r", 1000000, "keyspace length. The benchmark will expand the string __key__ inside an argument with a number in the specified range from 0 to keyspacelen-1. The substitution changes every time a command is executed.")
	datasize := flag.Uint64("d", 3, "Data size of the expanded string __data__ value in bytes. The benchmark will expand the string __data__ inside an argument with a charset with length specified by this parameter. The substitution changes every time a command is executed.")
	numberRequests := flag.Uint64("n", 10000000, "Total number of requests")
	debug := flag.Int("debug", 0, "Client debug level.")
	loop := flag.Bool("l", false, "Loop. Run the tests forever.")
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		log.Fatalf("You need to specify a command after the flag command arguments. The commands requires a minimum size of 2 ( command name and key )")
	}
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
	connectionStr := fmt.Sprintf("%s:%d", *host, *port)
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
	cluster := getOSSClusterConn(connectionStr, opts, *clients)
	for channel_id := 1; uint64(channel_id) <= *clients; channel_id++ {
		wg.Add(1)
		cmd := make([]string, len(args))
		copy(cmd, args)
		go ingestionRoutine(cluster, true, cmd, *keyspacelen, *datasize, samplesPerClient, *loop, int(*debug), &wg, keyPlaceOlderPos, dataPlaceOlderPos)
	}

	// listen for C-c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	tick := time.NewTicker(time.Duration(client_update_tick) * time.Second)
	closed, _, duration, totalMessages, _ := updateCLI(tick, c, *numberRequests, *loop)
	messageRate := float64(totalMessages) / float64(duration.Seconds())
	p50IngestionMs := float64(latencies.ValueAtQuantile(50.0)) / 1000.0
	p95IngestionMs := float64(latencies.ValueAtQuantile(95.0)) / 1000.0
	p99IngestionMs := float64(latencies.ValueAtQuantile(99.0)) / 1000.0

	fmt.Printf("\n")
	fmt.Printf("#################################################\n")
	fmt.Printf("Total Duration %.3f Seconds\n", duration.Seconds())
	fmt.Printf("Total Errors %d\n", totalErrors)
	fmt.Printf("Throughput summary: %.0f requests per second\n", messageRate)
	fmt.Printf("Latency summary (msec):\n")
	fmt.Printf("    %9s %9s %9s\n", "p50", "p95", "p99")
	fmt.Printf("    %9.3f %9.3f %9.3f\n", p50IngestionMs, p95IngestionMs, p99IngestionMs)

	if closed {
		return
	}

	// tell the goroutine to stop
	close(stopChan)
	// and wait for them both to reply back
	wg.Wait()
}

func updateCLI(tick *time.Ticker, c chan os.Signal, message_limit uint64, loop bool) (bool, time.Time, time.Duration, uint64, []float64) {

	start := time.Now()
	prevTime := time.Now()
	prevMessageCount := uint64(0)
	messageRateTs := []float64{}
	fmt.Printf("%26s %7s %25s %25s %7s %25s %25s\n", "Test time", " ", "Total Commands", "Total Errors", "", "Command Rate", "p50 lat. (msec)")
	for {
		select {
		case <-tick.C:
			{
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
