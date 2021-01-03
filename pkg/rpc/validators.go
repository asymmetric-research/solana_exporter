package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

type (
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
	}
)

func (c *RpcClient) GetVoteAccounts(ctx context.Context) (*GetVoteAccountsResponse, error) {
	body, err := c.rpcRequest(ctx, []byte(`{"jsonrpc":"2.0","id":1, "method":"getVoteAccounts", "params":[{"commitment":"recent"}]}`))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	var voteAccounts GetVoteAccountsResponse
	if err = json.Unmarshal(body, &voteAccounts); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return &voteAccounts, nil
}
