package main

import (
	"context"
	"fmt"
	"github.com/HdrHistogram/hdrhistogram-go"
	radix "github.com/mediocregopher/radix/v4"
	"golang.org/x/time/rate"
	"math"
	"math/rand"
	"strings"
	"sync"
)

var totalCommands uint64
var totalCached uint64
var totalErrors uint64
var totalCachedInvalidations uint64
var latencies *hdrhistogram.Histogram
var latenciesTick *hdrhistogram.Histogram
var benchmarkCommands arrayStringParameters
var benchmarkCommandsRatios arrayStringParameters

var cscInvalidationMutex sync.Mutex

const Inf = rate.Limit(math.MaxFloat64)
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type datapoint struct {
	success     bool
	duration_ms int64
	cachedEntry bool
}

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

type Client interface {

	// Do performs an Action on a Conn connected to the redis instance.
	Do(context.Context, radix.Action) error

	// Once Close() is called all future method calls on the Client will return
	// an error
	Close() error
}

func keyBuildLogic(keyPos int, dataPos int, datasize, keyspacelen uint64, cmdS []string, charset string) (newCmdS []string, key string) {
	newCmdS = make([]string, len(cmdS))
	copy(newCmdS, cmdS)
	if keyPos > -1 {
		keyV := fmt.Sprintf("%d", rand.Int63n(int64(keyspacelen)))
		key = strings.Replace(newCmdS[keyPos], "__key__", keyV, -1)
		newCmdS[keyPos] = key
	}
	if dataPos > -1 {
		newCmdS[dataPos] = stringWithCharset(int(datasize), charset)
	}
	return newCmdS, key
}

func getplaceholderpos(args []string, verbose bool) (keyPlaceOlderPos int, dataPlaceOlderPos int) {
	keyPlaceOlderPos = -1
	dataPlaceOlderPos = -1
	for pos, arg := range args {
		if arg == "__data__" {
			dataPlaceOlderPos = pos
		}

		if strings.Contains(arg, "__key__") {
			if verbose {
				fmt.Println(fmt.Sprintf("Detected __key__ placeholder in pos %d", pos))
			}
			keyPlaceOlderPos = pos
		}
	}
	return
}
