package rpc

import (
	"encoding/json"
	"fmt"
)

type (
	RPCError struct {
		Message string         `json:"message"`
		Code    int64          `json:"code"`
		Data    map[string]any `json:"data"`
	}

	response[T any] struct {
		jsonrpc string
		Result  T        `json:"result"`
		Error   RPCError `json:"error"`
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

	Block struct {
		BlockHeight       int64            `json:"blockHeight"`
		BlockTime         int64            `json:"blockTime,omitempty"`
		Blockhash         string           `json:"blockhash"`
		ParentSlot        int64            `json:"parentSlot"`
		PreviousBlockhash string           `json:"previousBlockhash"`
		Rewards           []BlockReward    `json:"rewards"`
		Transactions      []map[string]any `json:"transactions"`
	}

	BlockReward struct {
		Pubkey      string `json:"pubkey"`
		Lamports    int64  `json:"lamports"`
		PostBalance int64  `json:"postBalance"`
		RewardType  string `json:"rewardType"`
		Commission  uint8  `json:"commission"`
	}
)

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error (code: %d): %s (data: %v)", e.Code, e.Message, e.Data)
}

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
