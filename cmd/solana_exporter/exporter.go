package main

import (
	"context"
	"errors"
	"fmt"
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
	identity         string

	/// descriptors:
	ValidatorActive         *prometheus.Desc
	ValidatorActiveStake    *prometheus.Desc
	ValidatorLastVote       *prometheus.Desc
	ValidatorRootSlot       *prometheus.Desc
	ValidatorDelinquent     *prometheus.Desc
	AccountBalances         *prometheus.Desc
	NodeVersion             *prometheus.Desc
	NodeIsHealthy           *prometheus.Desc
	NodeNumSlotsBehind      *prometheus.Desc
	NodeMinimumLedgerSlot   *prometheus.Desc
	NodeFirstAvailableBlock *prometheus.Desc
}

func NewSolanaCollector(
	provider rpc.Provider, slotPace time.Duration, balanceAddresses, nodekeys, votekeys []string, identity string,
) *SolanaCollector {
	collector := &SolanaCollector{
		rpcClient:        provider,
		slotPace:         slotPace,
		balanceAddresses: CombineUnique(balanceAddresses, nodekeys, votekeys),
		identity:         identity,
		ValidatorActive: prometheus.NewDesc(
			"solana_validator_active",
			fmt.Sprintf(
				"Total number of active validators, grouped by %s ('%s' or '%s')",
				StateLabel, StateCurrent, StateDelinquent,
			),
			[]string{StateLabel},
			nil,
		),
		ValidatorActiveStake: prometheus.NewDesc(
			"solana_validator_active_stake",
			fmt.Sprintf("Active stake per validator (represented by %s and %s)", VotekeyLabel, NodekeyLabel),
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		ValidatorLastVote: prometheus.NewDesc(
			"solana_validator_last_vote",
			fmt.Sprintf("Last voted-on slot per validator (represented by %s and %s)", VotekeyLabel, NodekeyLabel),
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		ValidatorRootSlot: prometheus.NewDesc(
			"solana_validator_root_slot",
			fmt.Sprintf("Root slot per validator (represented by %s and %s)", VotekeyLabel, NodekeyLabel),
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		ValidatorDelinquent: prometheus.NewDesc(
			"solana_validator_delinquent",
			fmt.Sprintf("Whether a validator (represented by %s and %s) is delinquent", VotekeyLabel, NodekeyLabel),
			[]string{VotekeyLabel, NodekeyLabel},
			nil,
		),
		AccountBalances: prometheus.NewDesc(
			"solana_account_balance",
			fmt.Sprintf("Solana account balances, grouped by %s", AddressLabel),
			[]string{AddressLabel},
			nil,
		),
		NodeVersion: prometheus.NewDesc(
			"solana_node_version",
			"Node version of solana",
			[]string{VersionLabel},
			nil,
		),
		NodeIsHealthy: prometheus.NewDesc(
			"solana_node_is_healthy",
			fmt.Sprintf("Whether a node (%s) is healthy", IdentityLabel),
			[]string{IdentityLabel},
			nil,
		),
		NodeNumSlotsBehind: prometheus.NewDesc(
			"solana_node_num_slots_behind",
			fmt.Sprintf(
				"The number of slots that the node (%s) is behind the latest cluster confirmed slot.",
				IdentityLabel,
			),
			[]string{IdentityLabel},
			nil,
		),
		NodeMinimumLedgerSlot: prometheus.NewDesc(
			"solana_node_minimum_ledger_slot",
			fmt.Sprintf("The lowest slot that the node (%s) has information about in its ledger.", IdentityLabel),
			[]string{IdentityLabel},
			nil,
		),
		NodeFirstAvailableBlock: prometheus.NewDesc(
			"solana_node_first_available_block",
			fmt.Sprintf(
				"The slot of the lowest confirmed block that has not been purged from the node's (%s) ledger.",
				IdentityLabel,
			),
			[]string{IdentityLabel},
			nil,
		),
	}
	return collector
}

func (c *SolanaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.ValidatorActive
	ch <- c.NodeVersion
	ch <- c.ValidatorActiveStake
	ch <- c.ValidatorLastVote
	ch <- c.ValidatorRootSlot
	ch <- c.ValidatorDelinquent
	ch <- c.AccountBalances
	ch <- c.NodeIsHealthy
	ch <- c.NodeNumSlotsBehind
	ch <- c.NodeMinimumLedgerSlot
	ch <- c.NodeFirstAvailableBlock
}

func (c *SolanaCollector) collectVoteAccounts(ctx context.Context, ch chan<- prometheus.Metric) {
	voteAccounts, err := c.rpcClient.GetVoteAccounts(ctx, rpc.CommitmentConfirmed, nil)
	if err != nil {
		klog.Errorf("failed to get vote accounts: %v", err)
		ch <- prometheus.NewInvalidMetric(c.ValidatorActive, err)
		ch <- prometheus.NewInvalidMetric(c.ValidatorActiveStake, err)
		ch <- prometheus.NewInvalidMetric(c.ValidatorLastVote, err)
		ch <- prometheus.NewInvalidMetric(c.ValidatorRootSlot, err)
		ch <- prometheus.NewInvalidMetric(c.ValidatorDelinquent, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.ValidatorActive, prometheus.GaugeValue, float64(len(voteAccounts.Delinquent)), StateDelinquent,
	)
	ch <- prometheus.MustNewConstMetric(
		c.ValidatorActive, prometheus.GaugeValue, float64(len(voteAccounts.Current)), StateCurrent,
	)

	for _, account := range append(voteAccounts.Current, voteAccounts.Delinquent...) {
		ch <- prometheus.MustNewConstMetric(
			c.ValidatorActiveStake,
			prometheus.GaugeValue,
			float64(account.ActivatedStake),
			account.VotePubkey,
			account.NodePubkey,
		)
		ch <- prometheus.MustNewConstMetric(
			c.ValidatorLastVote,
			prometheus.GaugeValue,
			float64(account.LastVote),
			account.VotePubkey,
			account.NodePubkey,
		)
		ch <- prometheus.MustNewConstMetric(
			c.ValidatorRootSlot,
			prometheus.GaugeValue,
			float64(account.RootSlot),
			account.VotePubkey,
			account.NodePubkey,
		)
	}

	for _, account := range voteAccounts.Current {
		ch <- prometheus.MustNewConstMetric(
			c.ValidatorDelinquent, prometheus.GaugeValue, 0, account.VotePubkey, account.NodePubkey,
		)
	}
	for _, account := range voteAccounts.Delinquent {
		ch <- prometheus.MustNewConstMetric(
			c.ValidatorDelinquent, prometheus.GaugeValue, 1, account.VotePubkey, account.NodePubkey,
		)
	}
}

func (c *SolanaCollector) collectVersion(ctx context.Context, ch chan<- prometheus.Metric) {
	version, err := c.rpcClient.GetVersion(ctx)

	if err != nil {
		klog.Errorf("failed to get version: %v", err)
		ch <- prometheus.NewInvalidMetric(c.NodeVersion, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.NodeVersion, prometheus.GaugeValue, 1, version)
}
func (c *SolanaCollector) collectMinimumLedgerSlot(ctx context.Context, ch chan<- prometheus.Metric) {
	slot, err := c.rpcClient.GetMinimumLedgerSlot(ctx)

	if err != nil {
		klog.Errorf("failed to get minimum lidger slot: %v", err)
		ch <- prometheus.NewInvalidMetric(c.NodeMinimumLedgerSlot, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.NodeMinimumLedgerSlot, prometheus.GaugeValue, float64(*slot), c.identity)
}
func (c *SolanaCollector) collectFirstAvailableBlock(ctx context.Context, ch chan<- prometheus.Metric) {
	block, err := c.rpcClient.GetFirstAvailableBlock(ctx)

	if err != nil {
		klog.Errorf("failed to get first available block: %v", err)
		ch <- prometheus.NewInvalidMetric(c.NodeFirstAvailableBlock, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.NodeFirstAvailableBlock, prometheus.GaugeValue, float64(*block), c.identity)
}

func (c *SolanaCollector) collectBalances(ctx context.Context, ch chan<- prometheus.Metric) {
	balances, err := FetchBalances(ctx, c.rpcClient, c.balanceAddresses)
	if err != nil {
		klog.Errorf("failed to get balances: %v", err)
		ch <- prometheus.NewInvalidMetric(c.NodeVersion, err)
		return
	}

	for address, balance := range balances {
		ch <- prometheus.MustNewConstMetric(c.AccountBalances, prometheus.GaugeValue, balance, address)
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
				ch <- prometheus.NewInvalidMetric(c.NodeIsHealthy, err)
				ch <- prometheus.NewInvalidMetric(c.NodeNumSlotsBehind, err)
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
			ch <- prometheus.NewInvalidMetric(c.NodeIsHealthy, err)
			ch <- prometheus.NewInvalidMetric(c.NodeNumSlotsBehind, err)
			return
		}
	}

	ch <- prometheus.MustNewConstMetric(c.NodeIsHealthy, prometheus.GaugeValue, float64(isHealthy), c.identity)
	ch <- prometheus.MustNewConstMetric(c.NodeNumSlotsBehind, prometheus.GaugeValue, float64(numSlotsBehind), c.identity)

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
	collector := NewSolanaCollector(
		client, slotPacerSchedule, config.BalanceAddresses, config.NodeKeys, votekeys, identity,
	)
	slotWatcher := NewSlotWatcher(
		client, config.NodeKeys, votekeys, identity, config.ComprehensiveSlotTracking, config.MonitorBlockSizes,
	)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go slotWatcher.WatchSlots(ctx, collector.slotPace)

	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	klog.Infof("listening on %s", config.ListenAddress)
	klog.Fatal(http.ListenAndServe(config.ListenAddress, nil))
}
