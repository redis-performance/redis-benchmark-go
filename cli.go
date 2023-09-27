package main

import (
	shellwords "github.com/mattn/go-shellwords"
	"log"
	"math"
	"math/rand"
	"strconv"
)

type arrayStringParameters []string

func (i *arrayStringParameters) String() string {
	return "my string representation"
}

func (i *arrayStringParameters) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func sample(cdf []float32) int {
	r := rand.Float32()
	bucket := 0
	for r > cdf[bucket] {
		bucket++
	}
	return bucket
}

func prepareCommandsDistribution(queries arrayStringParameters, cmds [][]string, cmdRates []float64) (totalDifferentCommands int, cdf []float32) {
	totalDifferentCommands = len(cmds)
	var totalRateSum = 0.0
	var err error
	for i, rawCmdString := range queries {
		cmds[i], _ = shellwords.Parse(rawCmdString)
		if i >= len(benchmarkCommandsRatios) {
			cmdRates[i] = 1

		} else {
			cmdRates[i], err = strconv.ParseFloat(benchmarkCommandsRatios[i], 64)
			if err != nil {
				log.Fatalf("Error while converting query-rate param %s: %v", benchmarkCommandsRatios[i], err)
			}
		}
		totalRateSum += cmdRates[i]
	}
	// probability density function
	if math.Abs(1.0-totalRateSum) > 0.01 {
		log.Fatalf("Total ratio should be 1.0 ( currently is %f )", totalRateSum)
	}
	// probability density function
	if len(benchmarkCommandsRatios) > 0 && (len(benchmarkCommandsRatios) != (len(benchmarkCommands))) {
		log.Fatalf("When specifiying -cmd-ratio parameter, you need to have the same number of -cmd and -cmd-ratio parameters. Number of time -cmd ( %d ) != Number of times -cmd-ratio ( %d )", len(benchmarkCommands), len(benchmarkCommandsRatios))
	}
	pdf := make([]float32, len(queries))
	cdf = make([]float32, len(queries))
	for i := 0; i < len(cmdRates); i++ {
		pdf[i] = float32(cmdRates[i])
		cdf[i] = 0
	}
	// get cdf
	cdf[0] = pdf[0]
	for i := 1; i < len(cmdRates); i++ {
		cdf[i] = cdf[i-1] + pdf[i]
	}
	return
}
