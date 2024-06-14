package rpc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

/*
These tests are not to be run by CI and are by no means comprehensively indicative of a failure of this code.
However, they are a useful tool when developing the RPC package to make sure that everything is running at least
somewhat as expected
*/

var client = NewRPCClient("http://localhost:8899")

func TestClient_GetVersion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	version, err := client.GetVersion(ctx)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "1.", version[:2])
}

func TestClient_GetSlot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	slot, err := client.GetSlot(ctx)
	if err != nil {
		t.Error(err)
	}
	assert.Greater(t, slot, int64(271_760_309)) // slot at time of test creation
}

func TestClient_GetEpochInfo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	epochInfo, err := client.GetEpochInfo(ctx, CommitmentMax)
	if err != nil {
		t.Error(err)
	}
	// checks against time of creation:
	assert.Greater(t, epochInfo.Epoch, int64(628))
	assert.Greater(t, epochInfo.AbsoluteSlot, int64(271_760_309))
	assert.Greater(t, epochInfo.BlockHeight, int64(251_360_726))
	assert.Greater(t, epochInfo.TransactionCount, int64(295_561_750_573))

	// comparison checks:
	assert.Greater(t, epochInfo.AbsoluteSlot, epochInfo.BlockHeight)
	assert.Greater(t, epochInfo.SlotsInEpoch, epochInfo.SlotIndex)

	// unlikely to change any time soon
	assert.Equal(t, int64(432_000), epochInfo.SlotsInEpoch)
}

func TestClient_GetVoteAccounts(t *testing.T) {
	t.Run(
		"no_vote",
		func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			params := map[string]string{"commitment": string(CommitmentRecent)}
			voteAccounts, err := client.GetVoteAccounts(ctx, []interface{}{params})
			if err != nil {
				t.Error(err)
			}
			assert.Greater(t, len(voteAccounts.Current), 0)
			assert.Greater(t, len(voteAccounts.Delinquent), 0)
		},
	)

	t.Run(
		"with_vote",
		func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			params := map[string]string{
				"commitment": string(CommitmentRecent),
				"votePubkey": "CertusDeBmqN8ZawdkxK5kFGMwBXdudvWHYwtNgNhvLu",
			}
			voteAccounts, err := client.GetVoteAccounts(ctx, []interface{}{params})
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, len(voteAccounts.Current), 1)
			assert.Equal(t, len(voteAccounts.Delinquent), 0)
		},
	)
}

func TestClient_GetBlockProduction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	currentSlot, err := client.GetSlot(ctx)
	if err != nil {
		t.Error(err)
	}

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var nSlots int64 = 1000
	firstSlot := currentSlot - (2 * nSlots)
	lastSlot := currentSlot - nSlots
	blockProduction, err := client.GetBlockProduction(ctx, &firstSlot, &lastSlot)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, blockProduction.FirstSlot, firstSlot)
	assert.Equal(t, blockProduction.LastSlot, lastSlot)

	var totalLeaderSlots int64
	for _, hostProduction := range blockProduction.Hosts {
		assert.LessOrEqual(t, hostProduction.BlocksProduced, hostProduction.LeaderSlots)
		totalLeaderSlots += hostProduction.LeaderSlots
	}
	assert.Equal(t, totalLeaderSlots, nSlots+1)
}

// TODO: Add tests that make sure our error handling is correct
