package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"math"
	"math/rand"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

type (
	DynamicServer struct {
		Server *rpc.MockServer

		Slot             int
		BlockHeight      int
		Epoch            int
		EpochSize        int
		SlotTime         time.Duration
		TransactionCount int
		LeaderSchedule   map[string][]int
	}
)

var (
	nodekeys        = []string{"aaa", "bbb", "ccc"}
	votekeys        = []string{"AAA", "BBB", "CCC"}
	balances        = map[string]float64{"aaa": 1, "bbb": 2, "ccc": 3, "AAA": 4, "BBB": 5, "CCC": 6}
	rawVoteAccounts = map[string]any{
		"current": []map[string]any{
			{
				"activatedStake": 42,
				"lastVote":       147,
				"nodePubkey":     "bbb",
				"rootSlot":       18,
				"votePubkey":     "BBB",
			},
			{
				"activatedStake": 43,
				"lastVote":       148,
				"nodePubkey":     "ccc",
				"rootSlot":       19,
				"votePubkey":     "CCC",
			},
		},
		"delinquent": []map[string]any{
			{
				"activatedStake": 49,
				"lastVote":       92,
				"nodePubkey":     "aaa",
				"rootSlot":       3,
				"votePubkey":     "AAA",
			},
		},
	}
	rawBalances = map[string]int{
		"aaa": 1 * rpc.LamportsInSol,
		"bbb": 2 * rpc.LamportsInSol,
		"ccc": 3 * rpc.LamportsInSol,
		"AAA": 4 * rpc.LamportsInSol,
		"BBB": 5 * rpc.LamportsInSol,
		"CCC": 6 * rpc.LamportsInSol,
	}
	balanceMetricResponse = `
# HELP solana_account_balance Solana account balances, grouped by address
# TYPE solana_account_balance gauge
solana_account_balance{address="AAA"} 4
solana_account_balance{address="BBB"} 5
solana_account_balance{address="CCC"} 6
solana_account_balance{address="aaa"} 1
solana_account_balance{address="bbb"} 2
solana_account_balance{address="ccc"} 3
`
	dynamicLeaderSchedule = map[string][]int{
		"aaa": {0, 1, 2, 3, 12, 13, 14, 15},
		"bbb": {4, 5, 6, 7, 16, 17, 18, 19},
		"ccc": {8, 9, 10, 11, 20, 21, 22, 23},
	}
)

/*
===== DYNAMIC CLIENT =====:
*/

func voteTx(nodekey string) []string {
	return []string{nodekey, strings.ToUpper(nodekey), VoteProgram}
}

func NewDynamicRpcClient(t *testing.T, slot int) (*DynamicServer, *rpc.Client) {
	validatorInfos := make(map[string]rpc.MockValidatorInfo)
	for _, nodekey := range nodekeys {
		validatorInfos[nodekey] = rpc.MockValidatorInfo{
			Votekey:    strings.ToUpper(nodekey),
			Stake:      1_000_000,
			Delinquent: false,
		}
	}
	mockServer, client := rpc.NewMockClient(t,
		map[string]any{
			"getVersion":        map[string]string{"solana-core": "v1.0.0"},
			"getLeaderSchedule": dynamicLeaderSchedule,
			"getHealth":         "ok",
		},
		rawBalances,
		map[string]int{"AAA": 10, "BBB": 10, "CCC": 10},
		nil,
		validatorInfos,
	)
	server := DynamicServer{
		Slot:           0,
		Server:         mockServer,
		EpochSize:      24,
		SlotTime:       100 * time.Millisecond,
		LeaderSchedule: dynamicLeaderSchedule,
	}
	server.PopulateSlot(0)
	for {
		server.Slot++
		server.PopulateSlot(server.Slot)
		if server.Slot == slot {
			break
		}
	}
	return &server, client
}

func (c *DynamicServer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			c.Slot++
			c.PopulateSlot(c.Slot)
			// add 5% noise to the slot time:
			noiseRange := float64(c.SlotTime) * 0.05
			noise := (rand.Float64()*2 - 1) * noiseRange
			time.Sleep(c.SlotTime + time.Duration(noise))
		}
	}
}

func (c *DynamicServer) getLeader() string {
	index := c.Slot % c.EpochSize
	for leader, slots := range c.LeaderSchedule {
		if slices.Contains(slots, index) {
			return leader
		}
	}
	panic(fmt.Sprintf("leader not found at slot %d", c.Slot))
}

func (c *DynamicServer) PopulateSlot(slot int) {
	leader := c.getLeader()

	var block *rpc.MockBlockInfo
	// every 4th slot is skipped
	if slot%4 != 3 {
		c.BlockHeight++
		// only add some transactions if a block was produced
		transactions := [][]string{
			{"aaa", "bbb", "ccc"},
			{"xxx", "yyy", "zzz"},
		}
		// assume all validators voted
		for _, nodekey := range nodekeys {
			transactions = append(transactions, voteTx(nodekey))
			info := c.Server.GetValidatorInfo(nodekey)
			info.LastVote = slot
			c.Server.SetOpt(rpc.ValidatorInfoOpt, nodekey, info)
		}

		c.TransactionCount += len(transactions)
		block = &rpc.MockBlockInfo{Fee: 100, Transactions: transactions}
	}
	// add slot info:
	c.Server.SetOpt(rpc.SlotInfosOpt, slot, rpc.MockSlotInfo{Leader: leader, Block: block})

	// now update the server:
	c.Epoch = int(math.Floor(float64(slot) / float64(c.EpochSize)))
	c.Server.SetOpt(
		rpc.EasyResultsOpt,
		"getSlot",
		slot,
	)
	c.Server.SetOpt(
		rpc.EasyResultsOpt,
		"getEpochInfo",
		map[string]int{
			"absoluteSlot":     slot,
			"blockHeight":      c.BlockHeight,
			"epoch":            c.Epoch,
			"slotIndex":        slot % c.EpochSize,
			"slotsInEpoch":     c.EpochSize,
			"transactionCount": c.TransactionCount,
		},
	)
	c.Server.SetOpt(
		rpc.EasyResultsOpt,
		"minimumLedgerSlot",
		int(math.Max(0, float64(slot-c.EpochSize))),
	)
	c.Server.SetOpt(
		rpc.EasyResultsOpt,
		"getFirstAvailableBlock",
		int(math.Max(0, float64(slot-c.EpochSize))),
	)
}

/*
===== OTHER TEST UTILITIES =====:
*/

// extractName takes a Prometheus descriptor and returns its name
func extractName(desc *prometheus.Desc) string {
	// Get the string representation of the descriptor
	descString := desc.String()
	// Use regex to extract the metric name and help message from the descriptor string
	reName := regexp.MustCompile(`fqName: "([^"]+)"`)
	nameMatch := reName.FindStringSubmatch(descString)

	var name string
	if len(nameMatch) > 1 {
		name = nameMatch[1]
	}

	return name
}

type collectionTest struct {
	Name             string
	ExpectedResponse string
}

func runCollectionTests(t *testing.T, collector prometheus.Collector, testCases []collectionTest) {
	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			err := testutil.CollectAndCompare(collector, bytes.NewBufferString(test.ExpectedResponse), test.Name)
			assert.NoErrorf(t, err, "unexpected collecting result for %s: \n%s", test.Name, err)
		})
	}
}

func newTestConfig(fast bool) *ExporterConfig {
	pace := time.Duration(100) * time.Second
	if fast {
		pace = time.Duration(500) * time.Millisecond
	}
	config := ExporterConfig{
		time.Second * time.Duration(1),
		"http://localhost:8899",
		":8080",
		nodekeys,
		votekeys,
		nil,
		true,
		true,
		false,
		pace,
	}
	return &config
}

func TestSolanaCollector(t *testing.T) {
	_, client := NewDynamicRpcClient(t, 35)
	collector := NewSolanaCollector(client, newTestConfig(false))
	prometheus.NewPedanticRegistry().MustRegister(collector)

	testCases := []collectionTest{
		collector.ValidatorActiveStake.makeCollectionTest(abcValues(1_000_000, 1_000_000, 1_000_000)...),
		collector.ValidatorLastVote.makeCollectionTest(abcValues(34, 34, 34)...),
		collector.ValidatorRootSlot.makeCollectionTest(abcValues(0, 0, 0)...),
		collector.ValidatorDelinquent.makeCollectionTest(abcValues(0, 0, 0)...),
		{Name: "solana_account_balance", ExpectedResponse: balanceMetricResponse},
		collector.NodeVersion.makeCollectionTest(NewLV(1, "v1.0.0")),
		collector.NodeIsHealthy.makeCollectionTest(NewLV(1)),
		collector.NodeNumSlotsBehind.makeCollectionTest(NewLV(0)),
	}

	runCollectionTests(t, collector, testCases)
}
