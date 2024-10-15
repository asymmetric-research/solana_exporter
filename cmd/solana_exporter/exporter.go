package main

import (
	"context"
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

	/// descriptors:
	totalValidatorsDesc     *prometheus.Desc
	validatorActivatedStake *prometheus.Desc
	validatorLastVote       *prometheus.Desc
	validatorRootSlot       *prometheus.Desc
	validatorDelinquent     *prometheus.Desc
	solanaVersion           *prometheus.Desc
	balances                *prometheus.Desc
}

func NewSolanaCollector(
	provider rpc.Provider, slotPace time.Duration, balanceAddresses []string, nodekeys []string, votekeys []string,
) *SolanaCollector {
	collector := &SolanaCollector{
		rpcClient:        provider,
		slotPace:         slotPace,
		balanceAddresses: CombineUnique(balanceAddresses, nodekeys, votekeys),
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
}

func (c *SolanaCollector) collectVoteAccounts(ctx context.Context, ch chan<- prometheus.Metric) {
	voteAccounts, err := c.rpcClient.GetVoteAccounts(ctx, rpc.CommitmentConfirmed, nil)
	if err != nil {
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
		ch <- prometheus.NewInvalidMetric(c.solanaVersion, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.solanaVersion, prometheus.GaugeValue, 1, version)
}

func (c *SolanaCollector) collectBalances(ctx context.Context, ch chan<- prometheus.Metric) {
	balances, err := FetchBalances(ctx, c.rpcClient, c.balanceAddresses)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.solanaVersion, err)
		return
	}

	for address, balance := range balances {
		ch <- prometheus.MustNewConstMetric(c.balances, prometheus.GaugeValue, balance, address)
	}
}

func (c *SolanaCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.collectVoteAccounts(ctx, ch)
	c.collectVersion(ctx, ch)
	c.collectBalances(ctx, ch)
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

	collector := NewSolanaCollector(client, slotPacerSchedule, config.BalanceAddresses, config.NodeKeys, votekeys)
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
