package rpc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func newMethodTester(t *testing.T, method string, result any) (*MockServer, *Client) {
	return NewTestClient(t, map[string]any{method: result})
}

func TestClient_GetBalance(t *testing.T) {
	_, client := newMethodTester(t,
		"getBalance",
		map[string]any{"context": map[string]int{"slot": 1}, "value": 5 * LamportsInSol},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	balance, err := client.GetBalance(ctx, CommitmentFinalized, "")
	assert.NoError(t, err)
	assert.Equal(t, float64(5), balance)
}

func TestClient_GetBlockProduction(t *testing.T) {
	_, client := newMethodTester(t,
		"getBlockProduction",
		map[string]any{
			"context": map[string]int{
				"slot": 9887,
			},
			"value": map[string]any{
				"byIdentity": map[string][]int{
					"85iYT5RuzRTDgjyRa3cP8SYhM2j21fj7NhfJ3peu1DPr": {9888, 9886},
				},
				"range": map[string]int{
					"firstSlot": 0,
					"lastSlot":  9887,
				},
			},
		},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	blockProduction, err := client.GetBlockProduction(ctx, CommitmentFinalized, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t,
		&BlockProduction{
			ByIdentity: map[string]HostProduction{
				"85iYT5RuzRTDgjyRa3cP8SYhM2j21fj7NhfJ3peu1DPr": {9888, 9886},
			},
			Range: BlockProductionRange{
				FirstSlot: 0,
				LastSlot:  9887,
			},
		},
		blockProduction,
	)
}

func TestClient_GetEpochInfo(t *testing.T) {
	_, client := newMethodTester(t,
		"getEpochInfo",
		map[string]int{
			"absoluteSlot":     166_598,
			"blockHeight":      166_500,
			"epoch":            27,
			"slotIndex":        2_790,
			"slotsInEpoch":     8_192,
			"transactionCount": 22_661_093,
		},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	epochInfo, err := client.GetEpochInfo(ctx, CommitmentFinalized)
	assert.NoError(t, err)
	assert.Equal(t,
		EpochInfo{
			AbsoluteSlot:     166_598,
			BlockHeight:      166_500,
			Epoch:            27,
			SlotIndex:        2_790,
			SlotsInEpoch:     8_192,
			TransactionCount: 22_661_093,
		},
		*epochInfo,
	)
}

func TestClient_GetFirstAvailableBlock(t *testing.T) {
	_, client := newMethodTester(t, "getFirstAvailableBlock", 250_000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	block, err := client.GetFirstAvailableBlock(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 250_000, int(block))
}

func TestClient_GetHealth(t *testing.T) {
	_, client := newMethodTester(t, "getHealth", "ok")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	health, err := client.GetHealth(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "ok", health)
}

func TestClient_GetInflationReward(t *testing.T) {
	_, client := newMethodTester(t,
		"getInflationReward",
		[]map[string]int{
			{
				"amount":        2_500,
				"effectiveSlot": 224,
				"epoch":         2,
				"postBalance":   499_999_442_500,
			},
		},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inflationReward, err := client.GetInflationReward(ctx, CommitmentFinalized, nil, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t,
		[]InflationReward{
			{
				Amount:        2_500,
				EffectiveSlot: 224,
				Epoch:         2,
				PostBalance:   499_999_442_500,
			},
		},
		inflationReward,
	)
}

func TestClient_GetLeaderSchedule(t *testing.T) {
	expectedSchedule := map[string][]int64{
		"aaa": {0, 1, 2, 3, 4},
		"bbb": {5, 6, 7, 8, 9},
		"ccc": {10, 11, 12, 13, 14},
	}
	_, client := newMethodTester(t, "getLeaderSchedule", expectedSchedule)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	schedule, err := client.GetLeaderSchedule(ctx, CommitmentFinalized, 1)
	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, schedule)
}

func TestClient_GetMinimumLedgerSlot(t *testing.T) {
	_, client := newMethodTester(t, "minimumLedgerSlot", 250)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slot, err := client.GetMinimumLedgerSlot(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(250), slot)
}

func TestClient_GetSlot(t *testing.T) {
	_, client := newMethodTester(t, "getSlot", 1234)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slot, err := client.GetSlot(ctx, CommitmentFinalized)
	assert.NoError(t, err)
	assert.Equal(t, int64(1234), slot)
}

func TestClient_GetVersion(t *testing.T) {
	expectedResult := map[string]any{"feature-set": 2891131721, "solana-core": "1.16.7"}
	_, client := newMethodTester(t, "getVersion", expectedResult)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	version, err := client.GetVersion(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult["solana-core"], version)
}

func TestClient_GetVoteAccounts(t *testing.T) {
	_, client := newMethodTester(t,
		"getVoteAccounts",
		map[string]any{
			"current": []map[string]any{
				{
					"commission":       0,
					"epochVoteAccount": true,
					"epochCredits":     [][]int{{1, 64, 0}, {2, 192, 64}},
					"nodePubkey":       "B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",
					"lastVote":         147,
					"activatedStake":   42,
					"votePubkey":       "3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw",
				},
			},
			"delinquent": nil,
		},
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	voteAccounts, err := client.GetVoteAccounts(ctx, CommitmentFinalized, nil)
	assert.NoError(t, err)
	assert.Equal(t,
		&VoteAccounts{
			Current: []VoteAccount{
				{
					Commission:       0,
					EpochVoteAccount: true,
					EpochCredits:     [][]int{{1, 64, 0}, {2, 192, 64}},
					NodePubkey:       "B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",
					LastVote:         147,
					ActivatedStake:   42,
					VotePubkey:       "3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw",
				},
			},
		},
		voteAccounts,
	)
}
