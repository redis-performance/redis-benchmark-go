package main

import (
	"context"
	"github.com/mediocregopher/radix/v4"
	"log"
)

func getStandaloneConn(addr string, opts radix.Dialer, clients uint64) radix.Client {
	var err error
	var size int = int(clients)
	network := "tcp"
	ctx := context.Background()

	poolConfig := radix.PoolConfig{}
	poolConfig.Dialer = opts
	poolConfig.Size = size

	pool, err := poolConfig.New(ctx, network, addr)
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while creating new pool. error = %v", err)
	}
	return pool
}
