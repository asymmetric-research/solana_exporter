package rpc

type (
	GetBlockTimeResponse struct {
		Result int64    `json:"result"`
		Error  rpcError `json:"error"`
	}

	GetConfirmedBlocksResponse struct {
		Result []int64  `json:"result"`
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

	GetEpochInfoResponse struct {
		Result EpochInfo `json:"result"`
		Error  rpcError  `json:"error"`
	}

	LeaderSchedule map[string][]int64

	GetLeaderScheduleResponse struct {
		Result LeaderSchedule `json:"result"`
		Error  rpcError       `json:"error"`
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

	GetVoteAccountsResponse struct {
		Result struct {
			Current    []VoteAccount `json:"current"`
			Delinquent []VoteAccount `json:"delinquent"`
		} `json:"result"`
		Error rpcError `json:"error"`
	}

	GetVersionResponse struct {
		Result struct {
			Version string `json:"solana-core"`
		} `json:"result"`
		Error rpcError `json:"error"`
	}
)
