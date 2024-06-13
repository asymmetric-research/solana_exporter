package main

import (
	"context"
	"github.com/certusone/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"math/rand"
	"regexp"
	"time"
)

type (
	staticRPCClient struct{}
	// TODO: create dynamicRPCClient + according tests!
)

var (
	testValidators = []struct {
		identity string
		vote     string
	}{
		{"B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD", "3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"},
		{"C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD", "4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"},
		{"4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae", "xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"},
	}
	n               = len(testValidators)
	staticEpochInfo = rpc.EpochInfo{
		AbsoluteSlot:     166598,
		BlockHeight:      166500,
		Epoch:            27,
		SlotIndex:        2790,
		SlotsInEpoch:     8192,
		TransactionCount: 22661093,
	}
	staticBlockProduction = rpc.BlockProduction{
		FirstSlot: 1000,
		LastSlot:  2000,
		Hosts: map[string]rpc.BlockProductionPerHost{
			"B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD": {
				LeaderSlots:    400,
				BlocksProduced: 360,
			},
			"C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD": {
				LeaderSlots:    300,
				BlocksProduced: 296,
			},
			"4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae": {
				LeaderSlots:    300,
				BlocksProduced: 0,
			},
		},
	}
)

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetEpochInfo(ctx context.Context, commitment rpc.Commitment) (*rpc.EpochInfo, error) {
	return &staticEpochInfo, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetSlot(ctx context.Context) (int64, error) {
	return staticEpochInfo.AbsoluteSlot, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetVersion(ctx context.Context) (*string, error) {
	version := "1.16.7"
	return &version, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetVoteAccounts(
	ctx context.Context,
	params []interface{},
) (*rpc.VoteAccounts, error) {
	voteAccounts := rpc.VoteAccounts{
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
				NodePubkey:       "B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",
				RootSlot:         18,
				VotePubkey:       "3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw",
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
				NodePubkey:       "C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",
				RootSlot:         19,
				VotePubkey:       "4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw",
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
				NodePubkey:       "4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",
				RootSlot:         3,
				VotePubkey:       "xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU",
			},
		},
	}
	return &voteAccounts, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetBlockProduction(
	ctx context.Context,
	firstSlot *int64,
	lastSlot *int64,
) (rpc.BlockProduction, error) {
	return staticBlockProduction, nil
}

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

type (
	slotInfo struct {
		leader        string
		blockProduced bool
		votes         []string
	}
	dynamicRPCClient struct {
		slot             int
		blockHeight      int
		epoch            int
		epochSize        int
		transactionCount int
		version          string
		slotsInfo        map[int]slotInfo
		leaderIndex      int
	}
)

func (c *dynamicRPCClient) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		<-ticker.C

		c.newSlot()
	}
}

func (c *dynamicRPCClient) newSlot() {
	c.slot++

	// leader changes every 4 slots
	if c.slot%4 == 0 {
		c.leaderIndex = (c.leaderIndex + 1) % n
	}

	if c.slot%c.epochSize == 0 {
		c.epoch++
	}

	// assume 90% chance of block produced:
	blockProduced := rand.Intn(100) > 90
	if blockProduced {
		c.blockHeight++
		// only add some transactions if a block was produced
		c.transactionCount += rand.Intn(10)
	}

	// add slot info:
	c.slotsInfo[c.slot] = slotInfo{
		leader:        testValidators[c.leaderIndex].identity,
		blockProduced: blockProduced,
		// assume the other 2 validators voted:
		votes: []string{
			testValidators[(c.leaderIndex+1)%n].identity,
			testValidators[(c.leaderIndex+2)%n].identity,
		},
	}
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetEpochInfo(ctx context.Context, commitment rpc.Commitment) (*rpc.EpochInfo, error) {
	return &rpc.EpochInfo{
		AbsoluteSlot:     int64(c.slot),
		BlockHeight:      int64(c.blockHeight),
		Epoch:            int64(c.epoch),
		SlotIndex:        int64(c.slot % c.epochSize),
		SlotsInEpoch:     int64(c.epochSize),
		TransactionCount: int64(c.transactionCount),
	}, nil
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetSlot(ctx context.Context) (int64, error) {
	return int64(c.slot), nil
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetVersion(ctx context.Context) (*string, error) {
	return &c.version, nil
}

//goland:noinspection GoUnusedParameter
func (c *dynamicRPCClient) GetBlockProduction(
	ctx context.Context,
	firstSlot *int64,
	lastSlot *int64,
) (rpc.BlockProduction, error) {
	hostProduction := map[string]rpc.BlockProductionPerHost{
		testValidators[0].identity: {0, 0},
		testValidators[1].identity: {0, 0},
		testValidators[2].identity: {0, 0},
	}
	for i := *firstSlot; i <= *lastSlot; i++ {
		slotInfo := c.slotsInfo[int(i)]
		hp := hostProduction[slotInfo.leader]
		hp.LeaderSlots++
		if slotInfo.blockProduced {
			hp.BlocksProduced++
		}
		hostProduction[slotInfo.leader] = hp
	}
	return rpc.BlockProduction{
		FirstSlot: *firstSlot,
		LastSlot:  *lastSlot,
		Hosts:     hostProduction,
	}, nil
}
