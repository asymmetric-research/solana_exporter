package main

import (
	"bytes"
	"context"
	"github.com/certusone/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"regexp"
	"testing"
	"time"
)

type (
	staticRPCClient  struct{}
	dynamicRPCClient struct {
		Slot             int
		BlockHeight      int
		Epoch            int
		EpochSize        int
		SlotTime         time.Duration
		TransactionCount int
		Version          string
		SlotInfos        map[int]slotInfo
		LeaderIndex      int
		ValidatorInfos   map[string]validatorInfo
	}
	slotInfo struct {
		leader        string
		blockProduced bool
	}
	validatorInfo struct {
		Stake      int
		LastVote   int
		Commission int
		Delinquent bool
	}
)

var (
	identities      = []string{"aaa", "bbb", "ccc"}
	identityVotes   = map[string]string{"aaa": "AAA", "bbb": "BBB", "ccc": "CCC"}
	nv              = len(identities)
	staticEpochInfo = rpc.EpochInfo{
		AbsoluteSlot:     166598,
		BlockHeight:      166500,
		Epoch:            27,
		SlotIndex:        2790,
		SlotsInEpoch:     8192,
		TransactionCount: 22661093,
	}
	staticBlockProduction = rpc.BlockProduction{
		FirstSlot: 100000000,
		LastSlot:  200000000,
		Hosts: map[string]rpc.BlockProductionPerHost{
			"bbb": {LeaderSlots: 40000000, BlocksProduced: 36000000},
			"ccc": {LeaderSlots: 30000000, BlocksProduced: 29600000},
			"aaa": {LeaderSlots: 30000000, BlocksProduced: 10000000},
		},
	}
	staticVoteAccounts = rpc.VoteAccounts{
		Current: []rpc.VoteAccount{
			{
				ActivatedStake: 42,
				Commission:     0,
				EpochCredits: [][]int{
					{1, 64, 0},
					{2, 192, 64},
				},
				EpochVoteAccount: true,
				LastVote:         147,
				NodePubkey:       "bbb",
				RootSlot:         18,
				VotePubkey:       "BBB",
			},
			{
				ActivatedStake: 43,
				Commission:     1,
				EpochCredits: [][]int{
					{2, 65, 1},
					{3, 193, 65},
				},
				EpochVoteAccount: true,
				LastVote:         148,
				NodePubkey:       "ccc",
				RootSlot:         19,
				VotePubkey:       "CCC",
			},
		},
		Delinquent: []rpc.VoteAccount{
			{
				ActivatedStake: 49,
				Commission:     2,
				EpochCredits: [][]int{
					{10, 594, 6},
					{9, 98, 4},
				},
				EpochVoteAccount: true,
				LastVote:         92,
				NodePubkey:       "aaa",
				RootSlot:         3,
				VotePubkey:       "AAA",
			},
		},
	}
)

/*
===== STATIC CLIENT =====:
*/

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetEpochInfo(ctx context.Context, commitment rpc.Commitment) (*rpc.EpochInfo, error) {
	return &staticEpochInfo, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetSlot(ctx context.Context) (int64, error) {
	return staticEpochInfo.AbsoluteSlot, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetVersion(ctx context.Context) (string, error) {
	version := "1.16.7"
	return version, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetVoteAccounts(
	ctx context.Context,
	params []interface{},
) (*rpc.VoteAccounts, error) {
	return &staticVoteAccounts, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetBlockProduction(
	ctx context.Context,
	firstSlot *int64,
	lastSlot *int64,
) (rpc.BlockProduction, error) {
	return staticBlockProduction, nil
}

/*
===== DYNAMIC CLIENT =====:
*/

func newDynamicRPCClient() *dynamicRPCClient {
	validatorInfos := make(map[string]validatorInfo)
	for identity := range identityVotes {
		validatorInfos[identity] = validatorInfo{
			Stake:      1_000_000,
			LastVote:   0,
			Commission: 5,
			Delinquent: false,
		}
	}
	return &dynamicRPCClient{
		Slot:             0,
		BlockHeight:      0,
		Epoch:            0,
		EpochSize:        20,
		SlotTime:         100 * time.Millisecond,
		TransactionCount: 0,
		Version:          "v1.0.0",
		SlotInfos:        map[int]slotInfo{},
		LeaderIndex:      0,
		ValidatorInfos:   validatorInfos,
	}
}

func (c *dynamicRPCClient) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			c.newSlot()
			// add 5% noise to the slot time:
			noiseRange := float64(c.SlotTime) * 0.05
			noise := (rand.Float64()*2 - 1) * noiseRange
			time.Sleep(c.SlotTime + time.Duration(noise))
		}
	}
}

func (c *dynamicRPCClient) newSlot() {
	c.Slot++

	// leader changes every 4 slots
	if c.Slot%4 == 0 {
		c.LeaderIndex = (c.LeaderIndex + 1) % nv
	}

	if c.Slot%c.EpochSize == 0 {
		c.Epoch++
	}

	// assume 90% chance of block produced:
	blockProduced := rand.Intn(100) <= 90
	// add slot info:
	c.SlotInfos[c.Slot] = slotInfo{
		leader:        identities[c.LeaderIndex],
		blockProduced: blockProduced,
	}

	if blockProduced {
		c.BlockHeight++
		// only add some transactions if a block was produced
		c.TransactionCount += rand.Intn(10)
		// assume both other validators voted
		for i := 1; i < 3; i++ {
			otherValidatorIndex := (c.LeaderIndex + i) % nv
			identity := identities[otherValidatorIndex]
			info := c.ValidatorInfos[identity]
			info.LastVote = c.Slot
			c.ValidatorInfos[identity] = info
		}
	}
}

func (c *dynamicRPCClient) UpdateVersion(version string) {
	c.Version = version
}

func (c *dynamicRPCClient) UpdateStake(validator string, amount int) {
	info := c.ValidatorInfos[validator]
	info.Stake = amount
	c.ValidatorInfos[validator] = info
}

func (c *dynamicRPCClient) UpdateCommission(validator string, newCommission int) {
	info := c.ValidatorInfos[validator]
	info.Commission = newCommission
	c.ValidatorInfos[validator] = info
}

func (c *dynamicRPCClient) UpdateDelinquency(validator string, newDelinquent bool) {
	info := c.ValidatorInfos[validator]
	info.Delinquent = newDelinquent
	c.ValidatorInfos[validator] = info
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetEpochInfo(ctx context.Context, commitment rpc.Commitment) (*rpc.EpochInfo, error) {
	return &rpc.EpochInfo{
		AbsoluteSlot:     int64(c.Slot),
		BlockHeight:      int64(c.BlockHeight),
		Epoch:            int64(c.Epoch),
		SlotIndex:        int64(c.Slot % c.EpochSize),
		SlotsInEpoch:     int64(c.EpochSize),
		TransactionCount: int64(c.TransactionCount),
	}, nil
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetSlot(ctx context.Context) (int64, error) {
	return int64(c.Slot), nil
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetVersion(ctx context.Context) (string, error) {
	return c.Version, nil
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetVoteAccounts(
	ctx context.Context,
	params []interface{},
) (*rpc.VoteAccounts, error) {
	var currentVoteAccounts, delinquentVoteAccounts []rpc.VoteAccount
	for identity, vote := range identityVotes {
		info := c.ValidatorInfos[identity]
		voteAccount := rpc.VoteAccount{
			ActivatedStake:   int64(info.Stake),
			Commission:       info.Commission,
			EpochCredits:     [][]int{},
			EpochVoteAccount: true,
			LastVote:         info.LastVote,
			NodePubkey:       identity,
			RootSlot:         0,
			VotePubkey:       vote,
		}
		if info.Delinquent {
			delinquentVoteAccounts = append(delinquentVoteAccounts, voteAccount)
		} else {
			currentVoteAccounts = append(currentVoteAccounts, voteAccount)
		}
	}
	return &rpc.VoteAccounts{Current: currentVoteAccounts, Delinquent: delinquentVoteAccounts}, nil
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetBlockProduction(
	ctx context.Context,
	firstSlot *int64,
	lastSlot *int64,
) (rpc.BlockProduction, error) {
	hostProduction := make(map[string]rpc.BlockProductionPerHost)
	for _, identity := range identities {
		hostProduction[identity] = rpc.BlockProductionPerHost{LeaderSlots: 0, BlocksProduced: 0}
	}
	for i := *firstSlot; i <= *lastSlot; i++ {
		info := c.SlotInfos[int(i)]
		hp := hostProduction[info.leader]
		hp.LeaderSlots++
		if info.blockProduced {
			hp.BlocksProduced++
		}
		hostProduction[info.leader] = hp
	}
	return rpc.BlockProduction{
		FirstSlot: *firstSlot,
		LastSlot:  *lastSlot,
		Hosts:     hostProduction,
	}, nil
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
			assert.Nilf(t, "unexpected collecting result for %s: \n%s", test.Name, err)
		})
	}
}
