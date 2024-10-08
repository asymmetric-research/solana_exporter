package rpc

import (
	"encoding/json"
	"fmt"
)

type (
	response[T any] struct {
		jsonrpc string
		Result  T        `json:"result"`
		Error   rpcError `json:"error"`
		Id      int      `json:"id"`
	}

	contextualResult[T any] struct {
		Value   T `json:"value"`
		Context struct {
			Slot int64 `json:"slot"`
		} `json:"context"`
	}

	EpochInfo struct {
		// Current absolute slot in epoch
		AbsoluteSlot int64 `json:"absoluteSlot"`
		// Current block height
		BlockHeight int64 `json:"blockHeight"`
		// Current epoch number
		Epoch int64 `json:"epoch"`
		// Current slot relative to the start of the current epoch
		SlotIndex int64 `json:"slotIndex"`
		// Number of slots in this epoch
		SlotsInEpoch int64 `json:"slotsInEpoch"`
		// Total number of transactions
		TransactionCount int64 `json:"transactionCount"`
	}

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

	VoteAccounts struct {
		Current    []VoteAccount `json:"current"`
		Delinquent []VoteAccount `json:"delinquent"`
	}

	HostProduction struct {
		LeaderSlots    int64
		BlocksProduced int64
	}

	BlockProductionRange struct {
		FirstSlot int64 `json:"firstSlot"`
		LastSlot  int64 `json:"lastSlot"`
	}

	BlockProduction struct {
		ByIdentity map[string]HostProduction `json:"byIdentity"`
		Range      BlockProductionRange      `json:"range"`
	}

	InflationReward struct {
		Amount        int64 `json:"amount"`
		EffectiveSlot int64 `json:"effectiveSlot"`
		Epoch         int64 `json:"epoch"`
		PostBalance   int64 `json:"postBalance"`
	}
)

func (hp *HostProduction) UnmarshalJSON(data []byte) error {
	var arr []int64
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) != 2 {
		return fmt.Errorf("expected array of 2 integers, got %d", len(arr))
	}
	hp.LeaderSlots = arr[0]
	hp.BlocksProduced = arr[1]
	return nil
}

func (r response[T]) getError() rpcError {
	return r.Error
}

type HasRPCError interface {
	getError() rpcError
}
