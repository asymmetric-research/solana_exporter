package rpc

type (
	response[T any] struct {
		Result T        `json:"result"`
		Error  rpcError `json:"error"`
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

	blockProductionRange struct {
		FirstSlot int64  `json:"firstSlot"`
		LastSlot  *int64 `json:"lastSlot,omitempty"`
	}

	blockProductionResult struct {
		Value struct {
			ByIdentity map[string][]int64   `json:"byIdentity"`
			Range      blockProductionRange `json:"range"`
		} `json:"value"`
	}

	BlockProductionPerHost struct {
		LeaderSlots    int64
		BlocksProduced int64
	}

	BlockProduction struct {
		FirstSlot int64
		LastSlot  int64
		Hosts     map[string]BlockProductionPerHost
	}

	BalanceResult struct {
		Value   int64 `json:"value"`
		Context struct {
			Slot int64 `json:"slot"`
		} `json:"context"`
	}
)

func (r response[T]) getError() rpcError {
	return r.Error
}

type HasRPCError interface {
	getError() rpcError
}
