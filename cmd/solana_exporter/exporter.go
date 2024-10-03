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

var (
	httpTimeout         = 60 * time.Second
	rpcAddr             = flag.String("rpcURI", "", "Solana RPC URI (including protocol and path)")
	addr                = flag.String("addr", ":8080", "Listen address")
	votePubkey          = flag.String("votepubkey", "", "Validator vote address (will only return results of this address)")
	httpTimeoutSecs     = flag.Int("http_timeout", 60, "HTTP timeout in seconds")
	balanceAddresses    = flag.String("balance-addresses", "", "Comma-separated list of addresses to monitor balances")
	leaderSlotAddresses = flag.String(
		"leader-slot-addresses",
		"",
		"Comma-separated list of addresses to monitor leader slots by epoch for, leave nil to track by epoch for all validators (this creates a lot of Prometheus metrics with every new epoch).",
	)
)

func init() {
	klog.InitFlags(nil)
}

type solanaCollector struct {
	rpcClient rpc.Provider

	// config:
	slotPace         time.Duration
	balanceAddresses []string
	leaderSlotAddresses []string

	/// descriptors:
	totalValidatorsDesc     *prometheus.Desc
	validatorActivatedStake *prometheus.Desc
	validatorLastVote       *prometheus.Desc
	validatorRootSlot       *prometheus.Desc
	validatorDelinquent     *prometheus.Desc
	solanaVersion           *prometheus.Desc
	balances                *prometheus.Desc
}

func createSolanaCollector(
	provider rpc.Provider, slotPace time.Duration, balanceAddresses []string, leaderSlotAddresses []string,
) *solanaCollector {
	return &solanaCollector{
		rpcClient:           provider,
		slotPace:            slotPace,
		balanceAddresses:    balanceAddresses,
		leaderSlotAddresses: leaderSlotAddresses,
		totalValidatorsDesc: prometheus.NewDesc(
			"solana_active_validators",
			"Total number of active validators by state",
			[]string{"state"},
			nil,
		),
		validatorActivatedStake: prometheus.NewDesc(
			"solana_validator_activated_stake",
			"Activated stake per validator",
			[]string{"pubkey", "nodekey"},
			nil,
		),
		validatorLastVote: prometheus.NewDesc(
			"solana_validator_last_vote",
			"Last voted slot per validator",
			[]string{"pubkey", "nodekey"},
			nil,
		),
		validatorRootSlot: prometheus.NewDesc(
			"solana_validator_root_slot",
			"Root slot per validator",
			[]string{"pubkey", "nodekey"},
			nil,
		),
		validatorDelinquent: prometheus.NewDesc(
			"solana_validator_delinquent",
			"Whether a validator is delinquent",
			[]string{"pubkey", "nodekey"},
			nil,
		),
		solanaVersion: prometheus.NewDesc(
			"solana_node_version",
			"Node version of solana",
			[]string{"version"},
			nil,
		),
		balances: prometheus.NewDesc(
			"solana_account_balance",
			"Solana account balances",
			[]string{"address"},
			nil,
		),
	}
}

func NewSolanaCollector(rpcAddr string, balanceAddresses []string, leaderSlotAddresses []string) *solanaCollector {
	return createSolanaCollector(rpc.NewRPCClient(rpcAddr), slotPacerSchedule, balanceAddresses, leaderSlotAddresses)
}

func (c *solanaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalValidatorsDesc
	ch <- c.solanaVersion
	ch <- c.validatorActivatedStake
	ch <- c.validatorLastVote
	ch <- c.validatorRootSlot
	ch <- c.validatorDelinquent
	ch <- c.balances
}

func (c *solanaCollector) collectVoteAccounts(ctx context.Context, ch chan<- prometheus.Metric) {
	voteAccounts, err := c.rpcClient.GetVoteAccounts(ctx, rpc.CommitmentProcessed, votePubkey)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalValidatorsDesc, err)
		ch <- prometheus.NewInvalidMetric(c.validatorActivatedStake, err)
		ch <- prometheus.NewInvalidMetric(c.validatorLastVote, err)
		ch <- prometheus.NewInvalidMetric(c.validatorRootSlot, err)
		ch <- prometheus.NewInvalidMetric(c.validatorDelinquent, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.totalValidatorsDesc, prometheus.GaugeValue, float64(len(voteAccounts.Delinquent)), "delinquent",
	)
	ch <- prometheus.MustNewConstMetric(
		c.totalValidatorsDesc, prometheus.GaugeValue, float64(len(voteAccounts.Current)), "current",
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

func (c *solanaCollector) collectVersion(ctx context.Context, ch chan<- prometheus.Metric) {
	version, err := c.rpcClient.GetVersion(ctx)

	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.solanaVersion, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.solanaVersion, prometheus.GaugeValue, 1, version)
}

func (c *solanaCollector) collectBalances(ctx context.Context, ch chan<- prometheus.Metric) {
	balances, err := fetchBalances(ctx, c.rpcClient, c.balanceAddresses)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.solanaVersion, err)
		return
	}

	for address, balance := range balances {
		ch <- prometheus.MustNewConstMetric(c.balances, prometheus.GaugeValue, balance, address)
	}
}

func fetchBalances(ctx context.Context, client rpc.Provider, addresses []string) (map[string]float64, error) {
	balances := make(map[string]float64)
	for _, address := range addresses {
		balance, err := client.GetBalance(ctx, address)
		if err != nil {
			return nil, err
		}
		balances[address] = balance
	}
	return balances, nil
}

func (c *solanaCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	c.collectVoteAccounts(ctx, ch)
	c.collectVersion(ctx, ch)
	c.collectBalances(ctx, ch)
}

func main() {
	flag.Parse()

	if *rpcAddr == "" {
		klog.Fatal("Please specify -rpcURI")
	}

	if *leaderSlotAddresses == "" {
		klog.Warning(
			"Not specifying leader-slot-addresses will lead to potentially thousands of new " +
				"Prometheus metrics being created every epoch.",
		)
	}

	httpTimeout = time.Duration(*httpTimeoutSecs) * time.Second

	var (
		balAddresses []string
		lsAddresses  []string
	)
	if *balanceAddresses != "" {
		balAddresses = strings.Split(*balanceAddresses, ",")
	}
	if *leaderSlotAddresses != "" {
		lsAddresses = strings.Split(*leaderSlotAddresses, ",")
	}

	collector := NewSolanaCollector(*rpcAddr, balAddresses, lsAddresses)

	slotWatcher := SlotWatcher{client: collector.rpcClient}
	go slotWatcher.WatchSlots(context.Background(), collector.slotPace)

	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	klog.Infof("listening on %s", *addr)
	klog.Fatal(http.ListenAndServe(*addr, nil))
}
