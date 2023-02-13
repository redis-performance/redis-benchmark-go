package main

import (
	"context"
	"github.com/mediocregopher/radix/v4"
	"log"
)

func getOSSClusterConn(addr string, opts radix.Dialer, clients uint64) *radix.Cluster {
	var err error
	var vanillaCluster *radix.Cluster
	var size int = int(clients)
	ctx := context.Background()
	laddr := make([]string, 1)
	laddr[0] = addr
	poolConfig := radix.PoolConfig{}
	poolConfig.Dialer = opts
	poolConfig.Size = size

	clusterConfig := radix.ClusterConfig{}
	clusterConfig.PoolConfig = poolConfig

	vanillaCluster, err = clusterConfig.New(ctx, laddr)
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while creating new connection. error = %v", err)
	}
	// Issue CLUSTER SLOTS command
	err = vanillaCluster.Sync(ctx)
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while issuing CLUSTER SLOTS. error = %v", err)
	}
	return vanillaCluster
}
