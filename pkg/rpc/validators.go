package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
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

	VoteAccounts struct {
		Current    []VoteAccount `json:"current"`
		Delinquent []VoteAccount `json:"delinquent"`
	}

	GetVoteAccountsResponse struct {
		Result VoteAccounts `json:"result"`
		Error  rpcError1    `json:"error"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getvoteaccounts
func (c *Client) GetVoteAccounts(ctx context.Context, params []interface{}) (*VoteAccounts, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getVoteAccounts", params))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(3).Infof("getVoteAccounts response: %v", string(body))

	var resp GetVoteAccountsResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result, nil
}
