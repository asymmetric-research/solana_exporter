package main

import (
	"context"
	"flag"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strings"
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

var (
	httpTimeout = 60 * time.Second
	// general config:
	rpcUrl = flag.String(
		"rpc-url",
		"http://localhost:8899",
		"Solana RPC URL (including protocol and path), "+
			"e.g., 'http://localhost:8899' or 'https://api.mainnet-beta.solana.com'",
	)
	listenAddress = flag.String(
		"listen-address",
		":8080",
		"Listen address",
	)
	httpTimeoutSecs = flag.Int(
		"http-timeout",
		60,
		"HTTP timeout to use, in seconds.",
	)
	// parameters to specify what we're tracking:
	nodekeys = flag.String(
		"nodekeys",
		"",
		"Comma-separated list of nodekeys (identity accounts) representing validators to monitor.",
	)
	comprehensiveSlotTracking = flag.Bool(
		"comprehensive-slot-tracking",
		false,
		"Set this flag to track solana_leader_slots_by_epoch for ALL validators. "+
			"Warning: this will lead to potentially thousands of new Prometheus metrics being created every epoch.",
	)
	balanceAddresses = flag.String(
		"balance-addresses",
		"",
		"Comma-separated list of addresses to monitor SOL balances for, "+
			"in addition to the identity and vote accounts of the provided nodekeys.",
	)
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
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	c.collectVoteAccounts(ctx, ch)
	c.collectVersion(ctx, ch)
	c.collectBalances(ctx, ch)
}

func main() {
	ctx := context.Background()
	flag.Parse()

	if *comprehensiveSlotTracking {
		klog.Warning(
			"Comprehensive slot tracking will lead to potentially thousands of new " +
				"Prometheus metrics being created every epoch.",
		)
	}

	httpTimeout = time.Duration(*httpTimeoutSecs) * time.Second

	var (
		balAddresses      []string
		validatorNodekeys []string
	)
	if *balanceAddresses != "" {
		balAddresses = strings.Split(*balanceAddresses, ",")
	}
	if *nodekeys != "" {
		validatorNodekeys = strings.Split(*nodekeys, ",")
		klog.Infof("Monitoring the following validators: %v", validatorNodekeys)
	}

	client := rpc.NewRPCClient(*rpcUrl)
	ctx_, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	votekeys, err := GetAssociatedVoteAccounts(ctx_, client, rpc.CommitmentFinalized, validatorNodekeys)
	if err != nil {
		klog.Fatalf("Failed to get associated vote accounts for %v: %v", nodekeys, err)
	}
	collector := NewSolanaCollector(client, slotPacerSchedule, balAddresses, validatorNodekeys, votekeys)
	slotWatcher := NewSlotWatcher(client, validatorNodekeys, votekeys, *comprehensiveSlotTracking)
	go slotWatcher.WatchSlots(ctx, collector.slotPace)

	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	klog.Infof("listening on %s", *listenAddress)
	klog.Fatal(http.ListenAndServe(*listenAddress, nil))
}
