package main

import (
	"context"
	"github.com/certusone/solana_exporter/pkg/rpc"
)

type (
	staticRPCClient struct{}
	// TODO: create dynamicRPCClient + according tests!
)

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetEpochInfo(ctx context.Context, commitment rpc.Commitment) (*rpc.EpochInfo, error) {
	return &rpc.EpochInfo{
		AbsoluteSlot:     166598,
		BlockHeight:      166500,
		Epoch:            27,
		SlotIndex:        2790,
		SlotsInEpoch:     8192,
		TransactionCount: 22661093,
	}, nil
}

//goland:noinspection GoUnusedParameter
func (c *staticRPCClient) GetSlot(ctx context.Context) (int64, error) {
	return 166598, nil
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
	return rpc.BlockProduction{
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
	}, nil
}
