package main

import (
	"context"
	"github.com/certusone/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	httpTimeout = 2 * time.Second
)

var (
	listenAddr    = os.Getenv("LISTEN_ADDR")
	solanaRPCAddr = os.Getenv("SOLANA_RPC_ADDR")
)

func init() {
	if solanaRPCAddr == "" {
		log.Fatal("Please specify SOLANA_RPC_ADDR")
	}
	if listenAddr == "" {
		listenAddr = ":8080"
	}
}

type solanaCollector struct {
	rpcAddr string

	totalValidatorsDesc     *prometheus.Desc
	validatorActivatedStake *prometheus.Desc
	validatorLastVote       *prometheus.Desc
	validatorRootSlot       *prometheus.Desc
	validatorDelinquent     *prometheus.Desc
}

func NewSolanaCollector(rpcAddr string) prometheus.Collector {
	return &solanaCollector{
		rpcAddr: rpcAddr,
		totalValidatorsDesc: prometheus.NewDesc(
			"solana_active_validators",
			"Total number of active validators by state",
			[]string{"state"}, nil),
		validatorActivatedStake: prometheus.NewDesc(
			"solana_validator_activated_stake",
			"Activated stake per validator",
			[]string{"pubkey"}, nil),
		validatorLastVote: prometheus.NewDesc(
			"solana_validator_last_vote",
			"Last voted slot per validator",
			[]string{"pubkey"}, nil),
		validatorRootSlot: prometheus.NewDesc(
			"solana_validator_root_slot",
			"Root slot per validator",
			[]string{"pubkey"}, nil),
		validatorDelinquent: prometheus.NewDesc(
			"solana_validator_delinquent",
			"Whether a validator is delinquent",
			[]string{"pubkey"}, nil),
	}
}

func (collector solanaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.totalValidatorsDesc
}

func (collector solanaCollector) mustEmitMetrics(ch chan<- prometheus.Metric, response *rpc.GetVoteAccountsResponse) {
	ch <- prometheus.MustNewConstMetric(collector.totalValidatorsDesc, prometheus.GaugeValue,
		float64(len(response.Result.Delinquent)), "delinquent")
	ch <- prometheus.MustNewConstMetric(collector.totalValidatorsDesc, prometheus.GaugeValue,
		float64(len(response.Result.Current)), "current")

	for _, account := range append(response.Result.Current, response.Result.Delinquent...) {
		ch <- prometheus.MustNewConstMetric(collector.validatorActivatedStake, prometheus.GaugeValue,
			float64(account.ActivatedStake), account.VotePubkey)
		ch <- prometheus.MustNewConstMetric(collector.validatorLastVote, prometheus.GaugeValue,
			float64(account.LastVote), account.VotePubkey)
		ch <- prometheus.MustNewConstMetric(collector.validatorRootSlot, prometheus.GaugeValue,
			float64(account.RootSlot), account.VotePubkey)
	}
	for _, account := range response.Result.Current {
		ch <- prometheus.MustNewConstMetric(collector.validatorDelinquent, prometheus.GaugeValue,
			0, account.VotePubkey)
	}
	for _, account := range response.Result.Delinquent {
		ch <- prometheus.MustNewConstMetric(collector.validatorDelinquent, prometheus.GaugeValue,
			1, account.VotePubkey)
	}
}

func (collector solanaCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	accs, err := rpc.GetVoteAccounts(ctx, collector.rpcAddr)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(collector.totalValidatorsDesc, err)
		ch <- prometheus.NewInvalidMetric(collector.validatorActivatedStake, err)
		ch <- prometheus.NewInvalidMetric(collector.validatorLastVote, err)
		ch <- prometheus.NewInvalidMetric(collector.validatorRootSlot, err)
		ch <- prometheus.NewInvalidMetric(collector.validatorDelinquent, err)
	} else {
		collector.mustEmitMetrics(ch, accs)
	}
}

func main() {
	collector := NewSolanaCollector(solanaRPCAddr)
	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
