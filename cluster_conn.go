package main

import (
	"context"
	"github.com/mediocregopher/radix/v4"
	"log"
)

func getOSSClusterConn(addr string, dialer radix.Dialer, clients uint64) *radix.Cluster {
	var err error
	var vanillaCluster *radix.Cluster
	var size int = int(clients)
	ctx := context.Background()
	laddr := make([]string, 1)
	laddr[0] = addr
	poolConfig := radix.PoolConfig{}
	poolConfig.Dialer = dialer
	poolConfig.Size = size

	poolConfig.Dialer.CustomConn = func(ctx context.Context, network, addr string) (radix.Conn, error) {
		conn, err := dialer.Dial(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}

	clusterConfig := radix.ClusterConfig{}
	clusterConfig.PoolConfig = poolConfig

	vanillaCluster, err = clusterConfig.New(ctx, laddr)
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while creating new connection to %s. error = %v", addr, err)
	}
	// Issue CLUSTER SLOTS command
	err = vanillaCluster.Sync(ctx)
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while issuing CLUSTER SLOTS to %s. error = %v", addr, err)
	}
	return vanillaCluster
}
