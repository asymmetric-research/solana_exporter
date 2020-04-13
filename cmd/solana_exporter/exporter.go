package main

import (
	"bytes"
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type (
	VoteAccount struct {
		ActivatedStake   int64   `json:"activatedStake"`
		Commission       int     `json:"commission"`
		EpochCredits     [][]int `json:"epochCredits"`
		EpochVoteAccount bool    `json:"epochVoteAccount"`
		LastVote         int     `json:"lastVote"`
		NodePubkey       string  `json:"nodePubkey"`
		RootSlot         int     `json:"rootSlot"`
		VotePubkey       string  `json:"votePubkey"`
	}

	GetVoteAccountsResponse struct {
		Result struct {
			Current    []VoteAccount `json:"current"`
			Delinquent []VoteAccount `json:"delinquent"`
		} `json:"result"`
	}
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
	client  *http.Client
	rpcAddr string

	totalValidatorsDesc     *prometheus.Desc
	validatorActivatedStake *prometheus.Desc
	validatorLastVote       *prometheus.Desc
	validatorRootSlot       *prometheus.Desc
	validatorDelinquent     *prometheus.Desc
}

func NewSolanaCollector(rpcAddr string) prometheus.Collector {
	return &solanaCollector{
		client:  &http.Client{Timeout: httpTimeout},
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

func (collector solanaCollector) mustEmitMetrics(ch chan<- prometheus.Metric, response *GetVoteAccountsResponse) {
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
	var (
		voteAccounts GetVoteAccountsResponse
		body         []byte
		err          error
	)

	req, err := http.NewRequest("POST", collector.rpcAddr,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":1, "method":"getVoteAccounts", "params":[{"commitment":"recent"}]}`))
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := collector.client.Do(req)
	if err != nil {
		goto error
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		goto error
	}

	if err = json.Unmarshal(body, &voteAccounts); err != nil {
		goto error
	}

	collector.mustEmitMetrics(ch, &voteAccounts)
	return

error:
	ch <- prometheus.NewInvalidMetric(collector.totalValidatorsDesc, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorActivatedStake, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorLastVote, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorRootSlot, err)
	ch <- prometheus.NewInvalidMetric(collector.validatorDelinquent, err)
}

func main() {
	collector := NewSolanaCollector(solanaRPCAddr)
	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())
	panic(http.ListenAndServe(listenAddr, nil))
}
