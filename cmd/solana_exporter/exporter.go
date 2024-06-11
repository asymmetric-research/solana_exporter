package main

import (
	"context"
	"flag"
	"github.com/certusone/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

var (
	httpTimeout     = 60 * time.Second
	rpcAddr         = flag.String("rpcURI", "", "Solana RPC URI (including protocol and path)")
	addr            = flag.String("addr", ":8080", "Listen address")
	votePubkey      = flag.String("votepubkey", "", "Validator vote address (will only return results of this address)")
	httpTimeoutSecs = flag.Int("http_timeout", 60, "HTTP timeout in seconds")
)

func init() {
	klog.InitFlags(nil)
}

type solanaCollector struct {
	rpcClient *rpc.Client

	totalValidatorsDesc     *prometheus.Desc
	validatorActivatedStake *prometheus.Desc
	validatorLastVote       *prometheus.Desc
	validatorRootSlot       *prometheus.Desc
	validatorDelinquent     *prometheus.Desc
	solanaVersion           *prometheus.Desc
}

func NewSolanaCollector(rpcAddr string) *solanaCollector {
	return &solanaCollector{
		rpcClient: rpc.NewRPCClient(rpcAddr),
		totalValidatorsDesc: prometheus.NewDesc(
			"solana_active_validators",
			"Total number of active validators by state",
			[]string{"state"}, nil),
		validatorActivatedStake: prometheus.NewDesc(
			"solana_validator_activated_stake",
			"Activated stake per validator",
			[]string{"pubkey", "nodekey"}, nil),
		validatorLastVote: prometheus.NewDesc(
			"solana_validator_last_vote",
			"Last voted slot per validator",
			[]string{"pubkey", "nodekey"}, nil),
		validatorRootSlot: prometheus.NewDesc(
			"solana_validator_root_slot",
			"Root slot per validator",
			[]string{"pubkey", "nodekey"}, nil),
		validatorDelinquent: prometheus.NewDesc(
			"solana_validator_delinquent",
			"Whether a validator is delinquent",
			[]string{"pubkey", "nodekey"}, nil),
		solanaVersion: prometheus.NewDesc(
			"solana_node_version",
			"Node version of solana",
			[]string{"version"}, nil),
	}
}

func (c *solanaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalValidatorsDesc
	ch <- c.solanaVersion
}

func (c *solanaCollector) mustEmitMetrics(ch chan<- prometheus.Metric, response *rpc.GetVoteAccountsResponse) {
	ch <- prometheus.MustNewConstMetric(c.totalValidatorsDesc, prometheus.GaugeValue,
		float64(len(response.Result.Delinquent)), "delinquent")
	ch <- prometheus.MustNewConstMetric(c.totalValidatorsDesc, prometheus.GaugeValue,
		float64(len(response.Result.Current)), "current")

	for _, account := range append(response.Result.Current, response.Result.Delinquent...) {
		ch <- prometheus.MustNewConstMetric(c.validatorActivatedStake, prometheus.GaugeValue,
			float64(account.ActivatedStake), account.VotePubkey, account.NodePubkey)
		ch <- prometheus.MustNewConstMetric(c.validatorLastVote, prometheus.GaugeValue,
			float64(account.LastVote), account.VotePubkey, account.NodePubkey)
		ch <- prometheus.MustNewConstMetric(c.validatorRootSlot, prometheus.GaugeValue,
			float64(account.RootSlot), account.VotePubkey, account.NodePubkey)
	}
	for _, account := range response.Result.Current {
		ch <- prometheus.MustNewConstMetric(c.validatorDelinquent, prometheus.GaugeValue,
			0, account.VotePubkey, account.NodePubkey)
	}
	for _, account := range response.Result.Delinquent {
		ch <- prometheus.MustNewConstMetric(c.validatorDelinquent, prometheus.GaugeValue,
			1, account.VotePubkey, account.NodePubkey)
	}
}

func (c *solanaCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	params := map[string]string{"commitment": string(rpc.CommitmentRecent)}
	if *votePubkey != "" {
		params = map[string]string{"commitment": string(rpc.CommitmentRecent), "votePubkey": *votePubkey}
	}

	accs, err := c.rpcClient.GetVoteAccounts(ctx, []interface{}{params})
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalValidatorsDesc, err)
		ch <- prometheus.NewInvalidMetric(c.validatorActivatedStake, err)
		ch <- prometheus.NewInvalidMetric(c.validatorLastVote, err)
		ch <- prometheus.NewInvalidMetric(c.validatorRootSlot, err)
		ch <- prometheus.NewInvalidMetric(c.validatorDelinquent, err)
	} else {
		c.mustEmitMetrics(ch, accs)
	}

	version, err := c.rpcClient.GetVersion(ctx)

	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.solanaVersion, err)
	} else {
		ch <- prometheus.MustNewConstMetric(c.solanaVersion, prometheus.GaugeValue, 1, *version)
	}
}

func main() {
	flag.Parse()

	if *rpcAddr == "" {
		klog.Fatal("Please specify -rpcURI")
	}

	httpTimeout = time.Duration(*httpTimeoutSecs) * time.Second

	collector := NewSolanaCollector(*rpcAddr)

	go collector.WatchSlots()

	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	klog.Infof("listening on %s", *addr)
	klog.Fatal(http.ListenAndServe(*addr, nil))
}
