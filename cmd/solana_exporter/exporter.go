package main

import (
	"context"
	"errors"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

const (
	SkipStatusLabel = "status"
	StateLabel      = "state"
	NodekeyLabel    = "nodekey"
	VotekeyLabel    = "votekey"
	VersionLabel    = "version"
	AddressLabel    = "address"
	EpochLabel      = "epoch"
	IdentityLabel   = "identity"

	StatusSkipped = "skipped"
	StatusValid   = "valid"

	StateCurrent    = "current"
	StateDelinquent = "delinquent"
)

func init() {
	klog.InitFlags(nil)
}

type SolanaCollector struct {
	rpcClient rpc.Provider

	// config:
	slotPace         time.Duration
	balanceAddresses []string
	identity         *string

	/// descriptors:
	totalValidatorsDesc     *prometheus.Desc
	validatorActivatedStake *prometheus.Desc
	validatorLastVote       *prometheus.Desc
	validatorRootSlot       *prometheus.Desc
	validatorDelinquent     *prometheus.Desc
	solanaVersion           *prometheus.Desc
	balances                *prometheus.Desc
	isHealthy               *prometheus.Desc
	numSlotsBehind          *prometheus.Desc
	minimumLedgerSlot       *prometheus.Desc
	firstAvailableBlock     *prometheus.Desc
	blockHeight             *prometheus.Desc
}

func NewSolanaCollector(provider rpc.Provider, slotPace time.Duration, balanceAddresses []string, nodekeys []string, votekeys []string, identity *string) *SolanaCollector {
	collector := &SolanaCollector{
		rpcClient:        provider,
		slotPace:         slotPace,
		balanceAddresses: CombineUnique(balanceAddresses, nodekeys, votekeys),
		identity:         identity,
		totalValidatorsDesc: prometheus.NewDesc(
			"solana_active_validators",
			"Total number of active validators by state",
			[]string{StateLabel},
			nil,
		),
		validatorActivatedStake: prometheus.NewDesc(
			"solana_validator_activated_stake",
			"Activated stake per validator",
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		validatorLastVote: prometheus.NewDesc(
			"solana_validator_last_vote",
			"Last voted slot per validator",
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		validatorRootSlot: prometheus.NewDesc(
			"solana_validator_root_slot",
			"Root slot per validator",
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		validatorDelinquent: prometheus.NewDesc(
			"solana_validator_delinquent",
			"Whether a validator is delinquent",
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		solanaVersion: prometheus.NewDesc(
			"solana_node_version",
			"Node version of solana",
			[]string{VersionLabel},
			nil,
		),
		balances: prometheus.NewDesc(
			"solana_account_balance",
			"Solana account balances",
			[]string{AddressLabel},
			nil,
		),
		isHealthy: prometheus.NewDesc(
			"solana_is_healthy",
			"Whether the node is healthy or not.",
			[]string{IdentityLabel},
			nil,
		),
		numSlotsBehind: prometheus.NewDesc(
			"solana_num_slots_behind",
			"The number of slots that the node is behind the latest cluster confirmed slot.",
			[]string{IdentityLabel},
			nil,
		),
		minimumLedgerSlot: prometheus.NewDesc(
			"solana_minimum_ledger_slot",
			"The lowest slot that the node has information about in its ledger.",
			[]string{IdentityLabel},
			nil,
		),
		firstAvailableBlock: prometheus.NewDesc(
			"solana_first_available_block",
			"The slot of the lowest confirmed block that has not been purged from the ledger.",
			[]string{IdentityLabel},
			nil,
		),
		blockHeight: prometheus.NewDesc(
			"solana_block_height",
			"The current block height of the node.",
			[]string{IdentityLabel},
			nil,
		),
	}
	return collector
}

func (c *SolanaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalValidatorsDesc
	ch <- c.solanaVersion
	ch <- c.validatorActivatedStake
	ch <- c.validatorLastVote
	ch <- c.validatorRootSlot
	ch <- c.validatorDelinquent
	ch <- c.balances
	ch <- c.isHealthy
	ch <- c.numSlotsBehind
	ch <- c.minimumLedgerSlot
	ch <- c.firstAvailableBlock
	ch <- c.blockHeight
}

func (c *SolanaCollector) collectVoteAccounts(ctx context.Context, ch chan<- prometheus.Metric) {
	voteAccounts, err := c.rpcClient.GetVoteAccounts(ctx, rpc.CommitmentConfirmed, nil)
	if err != nil {
		klog.Errorf("failed to get vote accounts: %v", err)
		ch <- prometheus.NewInvalidMetric(c.totalValidatorsDesc, err)
		ch <- prometheus.NewInvalidMetric(c.validatorActivatedStake, err)
		ch <- prometheus.NewInvalidMetric(c.validatorLastVote, err)
		ch <- prometheus.NewInvalidMetric(c.validatorRootSlot, err)
		ch <- prometheus.NewInvalidMetric(c.validatorDelinquent, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.totalValidatorsDesc, prometheus.GaugeValue, float64(len(voteAccounts.Delinquent)), StateDelinquent,
	)
	ch <- prometheus.MustNewConstMetric(
		c.totalValidatorsDesc, prometheus.GaugeValue, float64(len(voteAccounts.Current)), StateCurrent,
	)

	for _, account := range append(voteAccounts.Current, voteAccounts.Delinquent...) {
		ch <- prometheus.MustNewConstMetric(
			c.validatorActivatedStake,
			prometheus.GaugeValue,
			float64(account.ActivatedStake),
			account.VotePubkey,
			account.NodePubkey,
		)
		ch <- prometheus.MustNewConstMetric(
			c.validatorLastVote,
			prometheus.GaugeValue,
			float64(account.LastVote),
			account.VotePubkey,
			account.NodePubkey,
		)
		ch <- prometheus.MustNewConstMetric(
			c.validatorRootSlot,
			prometheus.GaugeValue,
			float64(account.RootSlot),
			account.VotePubkey,
			account.NodePubkey,
		)
	}

	for _, account := range voteAccounts.Current {
		ch <- prometheus.MustNewConstMetric(
			c.validatorDelinquent, prometheus.GaugeValue, 0, account.VotePubkey, account.NodePubkey,
		)
	}
	for _, account := range voteAccounts.Delinquent {
		ch <- prometheus.MustNewConstMetric(
			c.validatorDelinquent, prometheus.GaugeValue, 1, account.VotePubkey, account.NodePubkey,
		)
	}
}

func (c *SolanaCollector) collectVersion(ctx context.Context, ch chan<- prometheus.Metric) {
	version, err := c.rpcClient.GetVersion(ctx)

	if err != nil {
		klog.Errorf("failed to get version: %v", err)
		ch <- prometheus.NewInvalidMetric(c.solanaVersion, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.solanaVersion, prometheus.GaugeValue, 1, version)
}
func (c *SolanaCollector) collectMinimumLedgerSlot(ctx context.Context, ch chan<- prometheus.Metric) {
	slot, err := c.rpcClient.GetMinimumLedgerSlot(ctx)

	if err != nil {
		klog.Errorf("failed to get minimum lidger slot: %v", err)
		ch <- prometheus.NewInvalidMetric(c.minimumLedgerSlot, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.minimumLedgerSlot, prometheus.GaugeValue, float64(*slot), *c.identity)
}
func (c *SolanaCollector) collectFirstAvailableBlock(ctx context.Context, ch chan<- prometheus.Metric) {
	block, err := c.rpcClient.GetFirstAvailableBlock(ctx)

	if err != nil {
		klog.Errorf("failed to get first available block: %v", err)
		ch <- prometheus.NewInvalidMetric(c.firstAvailableBlock, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.firstAvailableBlock, prometheus.GaugeValue, float64(*block), *c.identity)
}
func (c *SolanaCollector) collectBlockHeight(ctx context.Context, ch chan<- prometheus.Metric) {
	height, err := c.rpcClient.GetBlockHeight(ctx)

	if err != nil {
		klog.Errorf("failed to get block height: %v", err)
		ch <- prometheus.NewInvalidMetric(c.blockHeight, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.blockHeight, prometheus.GaugeValue, float64(*height), *c.identity)
}

func (c *SolanaCollector) collectBalances(ctx context.Context, ch chan<- prometheus.Metric) {
	balances, err := FetchBalances(ctx, c.rpcClient, c.balanceAddresses)
	if err != nil {
		klog.Errorf("failed to get balances: %v", err)
		ch <- prometheus.NewInvalidMetric(c.solanaVersion, err)
		return
	}

	for address, balance := range balances {
		ch <- prometheus.MustNewConstMetric(c.balances, prometheus.GaugeValue, balance, address)
	}
}

func (c *SolanaCollector) collectHealth(ctx context.Context, ch chan<- prometheus.Metric) {
	var (
		isHealthy      = 1
		numSlotsBehind int64
	)

	_, err := c.rpcClient.GetHealth(ctx)
	if err != nil {
		var rpcError *rpc.RPCError
		if errors.As(err, &rpcError) {
			var errorData rpc.NodeUnhealthyErrorData
			if rpcError.Data == nil {
				// if there is no data, then this is some unexpected error and should just be logged
				klog.Errorf("failed to get health: %v", err)
				ch <- prometheus.NewInvalidMetric(c.isHealthy, err)
				ch <- prometheus.NewInvalidMetric(c.numSlotsBehind, err)
				return
			}
			if err = rpc.UnpackRpcErrorData(rpcError, errorData); err != nil {
				// if we error here, it means we have the incorrect format
				klog.Fatalf("failed to unpack %s rpc error: %v", rpcError.Method, err.Error())
			}
			isHealthy = 0
			numSlotsBehind = errorData.NumSlotsBehind
		} else {
			// if it's not an RPC error, log it
			klog.Errorf("failed to get health: %v", err)
			ch <- prometheus.NewInvalidMetric(c.isHealthy, err)
			ch <- prometheus.NewInvalidMetric(c.numSlotsBehind, err)
			return
		}
	}

	ch <- prometheus.MustNewConstMetric(c.isHealthy, prometheus.GaugeValue, float64(isHealthy), *c.identity)
	ch <- prometheus.MustNewConstMetric(c.numSlotsBehind, prometheus.GaugeValue, float64(numSlotsBehind), *c.identity)

	return
}

func (c *SolanaCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.collectVoteAccounts(ctx, ch)
	c.collectVersion(ctx, ch)
	c.collectBalances(ctx, ch)
	c.collectHealth(ctx, ch)
	c.collectMinimumLedgerSlot(ctx, ch)
	c.collectFirstAvailableBlock(ctx, ch)
	c.collectBlockHeight(ctx, ch)
}

func main() {
	ctx := context.Background()

	config := NewExporterConfigFromCLI()
	if config.ComprehensiveSlotTracking {
		klog.Warning(
			"Comprehensive slot tracking will lead to potentially thousands of new " +
				"Prometheus metrics being created every epoch.",
		)
	}

	client := rpc.NewRPCClient(config.RpcUrl, config.HttpTimeout)
	votekeys, err := GetAssociatedVoteAccounts(ctx, client, rpc.CommitmentFinalized, config.NodeKeys)
	if err != nil {
		klog.Fatalf("Failed to get associated vote accounts for %v: %v", config.NodeKeys, err)
	}
	identity, err := client.GetIdentity(ctx)
	if err != nil {
		klog.Fatalf("Failed to get identity: %v", err)
	}
	collector := NewSolanaCollector(client, slotPacerSchedule, config.BalanceAddresses, config.NodeKeys, votekeys, identity)
	slotWatcher := NewSlotWatcher(
		client, config.NodeKeys, votekeys, config.ComprehensiveSlotTracking, config.MonitorBlockSizes,
	)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go slotWatcher.WatchSlots(ctx, collector.slotPace)

	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	klog.Infof("listening on %s", config.ListenAddress)
	klog.Fatal(http.ListenAndServe(config.ListenAddress, nil))
}
