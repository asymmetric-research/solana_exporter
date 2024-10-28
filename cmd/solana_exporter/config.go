package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/asymmetric-research/solana_exporter/pkg/slog"
	"time"
)

type (
	arrayFlags []string

	ExporterConfig struct {
		HttpTimeout               time.Duration
		RpcUrl                    string
		ListenAddress             string
		NodeKeys                  []string
		VoteKeys                  []string
		BalanceAddresses          []string
		ComprehensiveSlotTracking bool
		MonitorBlockSizes         bool
		LightMode                 bool
		SlotPace                  time.Duration
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
	ctx context.Context,
	httpTimeout time.Duration,
	rpcUrl string,
	listenAddress string,
	nodeKeys []string,
	balanceAddresses []string,
	comprehensiveSlotTracking bool,
	monitorBlockSizes bool,
	lightMode bool,
	slotPace time.Duration,
) (*ExporterConfig, error) {
	logger := slog.Get()
	logger.Infow(
		"Setting up export config with ",
		"httpTimeout", httpTimeout.Seconds(),
		"rpcUrl", rpcUrl,
		"listenAddress", listenAddress,
		"nodeKeys", nodeKeys,
		"balanceAddresses", balanceAddresses,
		"comprehensiveSlotTracking", comprehensiveSlotTracking,
		"monitorBlockSizes", monitorBlockSizes,
		"lightMode", lightMode,
	)
	if lightMode && (monitorBlockSizes || comprehensiveSlotTracking) {
		return nil, fmt.Errorf("-light-mode is not compatible with -comprehensiveSlotTracking or -monitorBlockSizes")
	}

	// get votekeys from rpc:
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	client := rpc.NewRPCClient(rpcUrl, httpTimeout)
	voteKeys, err := GetAssociatedVoteAccounts(ctx, client, rpc.CommitmentFinalized, nodeKeys)
	if err != nil {
		return nil, fmt.Errorf("error getting vote accounts: %w", err)
	}

	config := ExporterConfig{
		HttpTimeout:               httpTimeout,
		RpcUrl:                    rpcUrl,
		ListenAddress:             listenAddress,
		NodeKeys:                  nodeKeys,
		VoteKeys:                  voteKeys,
		BalanceAddresses:          balanceAddresses,
		ComprehensiveSlotTracking: comprehensiveSlotTracking,
		MonitorBlockSizes:         monitorBlockSizes,
		LightMode:                 lightMode,
		SlotPace:                  slotPace,
	}
	return &config, nil
}

func NewExporterConfigFromCLI(ctx context.Context) (*ExporterConfig, error) {
	var (
		httpTimeout               int
		rpcUrl                    string
		listenAddress             string
		nodekeys                  arrayFlags
		balanceAddresses          arrayFlags
		comprehensiveSlotTracking bool
		monitorBlockSizes         bool
		lightMode                 bool
		slotPace                  int
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
	flag.BoolVar(
		&lightMode,
		"light-mode",
		false,
		"Set this flag to enable light-mode. In light mode, only metrics specific to the node being queried "+
			"are reported (i.e., metrics such as inflation rewards which are visible from any RPC node, "+
			"are not reported).",
	)
	flag.IntVar(
		&slotPace,
		"slot-pace",
		1,
		"This is the time between slot-watching metric collections, defaults to 1s.",
	)
	flag.Parse()

	config, err := NewExporterConfig(
		ctx,
		time.Duration(httpTimeout)*time.Second,
		rpcUrl,
		listenAddress,
		nodekeys,
		balanceAddresses,
		comprehensiveSlotTracking,
		monitorBlockSizes,
		lightMode,
		time.Duration(slotPace)*time.Second,
	)
	if err != nil {
		return nil, err
	}
	return config, nil
}
