package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"

	"github.com/HdrHistogram/hdrhistogram-go"
	radix "github.com/mediocregopher/radix/v4"
	"golang.org/x/time/rate"
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

func stringWithCharset(length int, charset string, rng *rand.Rand, dataCache map[int]string) string {
	if length <= 0 {
		return ""
	}

	// For common sizes and default charset, use per-goroutine cache
	if charset == "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" && length <= 1024 {
		if cached, exists := dataCache[length]; exists {
			return cached
		}

		// Generate and cache
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rng.Intn(len(charset))]
		}
		result := string(b)
		dataCache[length] = result
		return result
	}

	// For small lengths, use the original method
	if length <= 64 {
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rng.Intn(len(charset))]
		}
		return string(b)
	}

	// For larger lengths, use a more efficient approach
	// Generate chunks of random data and repeat pattern
	chunkSize := 64
	if length < 256 {
		chunkSize = 32
	}

	b := make([]byte, length)
	charsetLen := len(charset)

	// Generate initial random chunk
	for i := 0; i < chunkSize && i < length; i++ {
		b[i] = charset[rng.Intn(charsetLen)]
	}

	// For remaining bytes, use pattern repetition with some randomness
	for i := chunkSize; i < length; i += chunkSize {
		end := i + chunkSize
		if end > length {
			end = length
		}

		// Copy previous chunk with slight variation
		for j := i; j < end; j++ {
			if rng.Intn(8) == 0 { // 12.5% chance to randomize
				b[j] = charset[rng.Intn(charsetLen)]
			} else {
				b[j] = b[j-chunkSize]
			}
		}
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

func keyBuildLogic(keyPos int, dataPos int, datasize, keyspacelen uint64, cmdS []string, charset string, rng *rand.Rand, dataCache map[int]string) (newCmdS []string, key string) {
	newCmdS = make([]string, len(cmdS))
	copy(newCmdS, cmdS)
	if keyPos > -1 {
		var keyV string
		if keyspacelen == 1 {
			keyV = "0" // Optimize for single key case
		} else {
			keyV = fmt.Sprintf("%d", rng.Int63n(int64(keyspacelen)))
		}
		key = strings.Replace(newCmdS[keyPos], "__key__", keyV, -1)
		newCmdS[keyPos] = key
	}
	if dataPos > -1 {
		newCmdS[dataPos] = stringWithCharset(int(datasize), charset, rng, dataCache)
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
