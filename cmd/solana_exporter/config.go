package main

import (
	"flag"
	"fmt"
	"time"
)

type (
	arrayFlags []string

	ExporterConfig struct {
		HttpTimeout               time.Duration
		RpcUrl                    string
		ListenAddress             string
		NodeKeys                  []string
		BalanceAddresses          []string
		ComprehensiveSlotTracking bool
		MonitorBlockSizes         bool
	}
)

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func NewExporterConfig(
	httpTimeout int,
	rpcUrl string,
	listenAddress string,
	nodeKeys []string,
	balanceAddresses []string,
	comprehensiveSlotTracking bool,
	monitorBlockSizes bool,
) *ExporterConfig {
	return &ExporterConfig{
		HttpTimeout:               time.Duration(httpTimeout) * time.Second,
		RpcUrl:                    rpcUrl,
		ListenAddress:             listenAddress,
		NodeKeys:                  nodeKeys,
		BalanceAddresses:          balanceAddresses,
		ComprehensiveSlotTracking: comprehensiveSlotTracking,
		MonitorBlockSizes:         monitorBlockSizes,
	}
}

func NewExporterConfigFromCLI() *ExporterConfig {
	var (
		httpTimeout               int
		rpcUrl                    string
		listenAddress             string
		nodekeys                  arrayFlags
		balanceAddresses          arrayFlags
		comprehensiveSlotTracking bool
		monitorBlockSizes         bool
	)
	flag.IntVar(
		&httpTimeout,
		"http-timeout",
		60,
		"HTTP timeout to use, in seconds.",
	)
	flag.StringVar(
		&rpcUrl,
		"rpc-url",
		"http://localhost:8899",
		"Solana RPC URL (including protocol and path), "+
			"e.g., 'http://localhost:8899' or 'https://api.mainnet-beta.solana.com'",
	)
	flag.StringVar(
		&listenAddress,
		"listen-address",
		":8080",
		"Listen address",
	)
	flag.Var(
		&nodekeys,
		"nodekey",
		"Solana nodekey (identity account) representing validator to monitor - can set multiple times.",
	)
	flag.Var(
		&balanceAddresses,
		"balance-address",
		"Address to monitor SOL balances for, in addition to the identity and vote accounts of the "+
			"provided nodekeys - can be set multiple times.",
	)
	flag.BoolVar(
		&comprehensiveSlotTracking,
		"comprehensive-slot-tracking",
		false,
		"Set this flag to track solana_leader_slots_by_epoch for ALL validators. "+
			"Warning: this will lead to potentially thousands of new Prometheus metrics being created every epoch.",
	)
	flag.BoolVar(
		&monitorBlockSizes,
		"monitor-block-sizes",
		false,
		"Set this flag to track block sizes (number of transactions) for the configured validators. "+
			"Warning: this might grind the RPC node.",
	)
	flag.Parse()

	return NewExporterConfig(
		httpTimeout, rpcUrl, listenAddress, nodekeys, balanceAddresses, comprehensiveSlotTracking, monitorBlockSizes,
	)
}
