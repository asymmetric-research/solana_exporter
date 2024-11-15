package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"github.com/asymmetric-research/solana-exporter/pkg/rpc"
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
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
		map[string]any{
			"getLeaderSchedule": map[string]any{
				"aaa": []int{0, 3, 6, 9, 12},
				"bbb": []int{1, 4, 7, 10, 13},
				"ccc": []int{2, 5, 8, 11, 14},
			},
		},
		nil, nil, nil, nil,
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
	simulator, client := NewSimulator(t, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetchedBalances, err := FetchBalances(ctx, client, CombineUnique(simulator.Nodekeys, simulator.Votekeys))
	assert.NoError(t, err)
	assert.Equal(t,
		map[string]float64{"aaa": 1, "bbb": 2, "ccc": 3, "AAA": 4, "BBB": 5, "CCC": 6},
		fetchedBalances,
	)
}

func TestGetAssociatedVoteAccounts(t *testing.T) {
	simulator, client := NewSimulator(t, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voteAccounts, err := GetAssociatedVoteAccounts(ctx, client, rpc.CommitmentFinalized, simulator.Nodekeys)
	assert.NoError(t, err)
	assert.Equal(t, simulator.Votekeys, voteAccounts)
}

func TestGetEpochBounds(t *testing.T) {
	epoch := rpc.EpochInfo{AbsoluteSlot: 25, SlotIndex: 5, SlotsInEpoch: 10}
	first, last := GetEpochBounds(&epoch)
	assert.Equal(t, int64(20), first)
	assert.Equal(t, int64(29), last)
}

//go:embed testdata/block-297609329.json
var blockJson []byte

func TestCountVoteTransactions(t *testing.T) {
	var block rpc.Block
	err := json.Unmarshal(blockJson, &block)
	assert.NoError(t, err)

	voteCount, err := CountVoteTransactions(&block)
	assert.NoError(t, err)
	// https://explorer.solana.com/block/297609329
	assert.Equal(t, 1048, voteCount)
	assert.Equal(t, 446, len(block.Transactions)-voteCount)
}

func TestEpochTrackedValidators_GetTrackedValidators(t *testing.T) {
	etv := EpochTrackedValidators{
		trackedNodekeys: map[int64]map[string]struct{}{
			1: {"a": struct{}{}, "b": struct{}{}},
			2: {"c": struct{}{}, "d": struct{}{}},
		},
	}
	t.Run(
		"simple_get",
		func(t *testing.T) {
			nodekeys, err := etv.GetTrackedValidators(1)
			assert.NoError(t, err)
			sort.Strings(nodekeys)
			assert.Equal(t, []string{"a", "b"}, nodekeys)
		},
	)
	t.Run(
		"second-get",
		func(t *testing.T) {
			nodekeys, err := etv.GetTrackedValidators(2)
			assert.NoError(t, err)
			sort.Strings(nodekeys)
			assert.Equal(t, []string{"c", "d"}, nodekeys)
		},
	)
	t.Run(
		"failed-get",
		func(t *testing.T) {
			_, err := etv.GetTrackedValidators(1)
			assert.Error(t, err)
		},
	)

}

func TestEpochTrackedValidators_AddTrackedValidators(t *testing.T) {
	etv := NewEpochTrackedValidators()
	etv.AddTrackedNodekeys(1, []string{"a", "b"})
	etv.AddTrackedNodekeys(2, []string{"c", "d"})
	assert.Equal(t,
		map[int64]map[string]struct{}{
			1: {"a": struct{}{}, "b": struct{}{}},
			2: {"c": struct{}{}, "d": struct{}{}},
		},
		etv.trackedNodekeys,
	)
}
