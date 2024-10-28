package main

import (
	"context"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	rawLeaderSchedule = map[string]any{
		"aaa": []int{0, 3, 6, 9, 12},
		"bbb": []int{1, 4, 7, 10, 13},
		"ccc": []int{2, 5, 8, 11, 14},
	}
)

func TestSelectFromSchedule(t *testing.T) {
	selected := SelectFromSchedule(
		map[string][]int64{
			"aaa": {0, 3, 6, 9, 12},
			"bbb": {1, 4, 7, 10, 13},
			"ccc": {2, 5, 8, 11, 14},
		},
		5,
		10,
	)
	assert.Equal(t,
		map[string][]int64{"aaa": {6, 9}, "bbb": {7, 10}, "ccc": {5, 8}},
		selected,
	)
}

func TestGetTrimmedLeaderSchedule(t *testing.T) {
	_, client := rpc.NewMockClient(t,
		map[string]any{"getLeaderSchedule": rawLeaderSchedule}, nil, nil, nil, nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	schedule, err := GetTrimmedLeaderSchedule(ctx, client, []string{"aaa", "bbb"}, 10, 10)
	assert.NoError(t, err)
	assert.Equal(t, map[string][]int64{"aaa": {10, 13, 16, 19, 22}, "bbb": {11, 14, 17, 20, 23}}, schedule)
}

func TestCombineUnique(t *testing.T) {
	var (
		v1 = []string{"1", "2", "3"}
		v2 = []string{"2", "3", "4"}
		v3 = []string{"3", "4", "5"}
	)

	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, CombineUnique(v1, v2, v3))
	assert.Equal(t, []string{"2", "3", "4", "5"}, CombineUnique(nil, v2, v3))
	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, CombineUnique(v1, nil, v3))

}

func TestFetchBalances(t *testing.T) {
	_, client := rpc.NewMockClient(t, nil, rawBalances, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetchedBalances, err := FetchBalances(ctx, client, CombineUnique(nodekeys, votekeys))
	assert.NoError(t, err)
	assert.Equal(t, balances, fetchedBalances)
}

func TestGetAssociatedVoteAccounts(t *testing.T) {
	_, client := rpc.NewMockClient(t,
		map[string]any{"getVoteAccounts": rawVoteAccounts},
		nil,
		nil,
		nil,
		nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voteAccounts, err := GetAssociatedVoteAccounts(ctx, client, rpc.CommitmentFinalized, nodekeys)
	assert.NoError(t, err)
	assert.Equal(t, votekeys, voteAccounts)
}

func TestGetEpochBounds(t *testing.T) {
	epoch := rpc.EpochInfo{AbsoluteSlot: 25, SlotIndex: 5, SlotsInEpoch: 10}
	first, last := GetEpochBounds(&epoch)
	assert.Equal(t, int64(20), first)
	assert.Equal(t, int64(29), last)
}
