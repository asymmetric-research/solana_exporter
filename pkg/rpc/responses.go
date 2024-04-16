package rpc

type (
	rpcError struct {
		Message string `json:"message"`
		Code    int64  `json:"id"`
	}

	Response[T any] struct {
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
		// Total number of transactions ever (?)
		TransactionCount int64 `json:"transactionCount"`
	}

	LeaderSchedule map[string][]int64

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

	VersionInfo struct {
		Version string `json:"solana-core"`
	}
)

func (r Response[T]) getError() rpcError {
	return r.Error
}

type HasRPCError interface {
	getError() rpcError
}
