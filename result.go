package main

type testResult struct {
	StartTime      int64     `json:"StartTime"`
	Duration       float64   `json:"Duration"`
	CommandRate    float64   `json:"CommandRate"`
	TotalCommands  uint64    `json:"TotalCommands"`
	CommandRateTs  []float64 `json:"CommandRateTs"`
	P50LatenciesTs []float64 `json:"P50LatenciesTs"`
}
