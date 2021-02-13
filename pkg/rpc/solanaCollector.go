package rpc

import "github.com/prometheus/client_golang/prometheus"

type SolanaCollector struct {
	rpcClient               *RPCClient
	totalValidatorsDesc     *prometheus.Desc
	validatorActivatedStake *prometheus.Desc
	validatorLastVote       *prometheus.Desc
	validatorRootSlot       *prometheus.Desc
	validatorDelinquent     *prometheus.Desc
	transactionCount        *prometheus.Desc
}
