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
		Balance          int64   `json:"balance"`
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

	GetBalanceResponse struct {
		Result struct {
			Value int `json:"value"`
		} `json:"result"`
		Error rpcError `json:"error"`
	}
)

// https://docs.velas.com/apps/jsonrpc-api/#getbalance
func (c *RPCClient) GetBalance(ctx context.Context, nodePubkey string) (*GetBalanceResponse, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getBalance", []interface{}{nodePubkey}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(3).Infof("getBalance response: %v", string(body))

	var resp GetBalanceResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}

// https://docs.solana.com/developing/clients/jsonrpc-api#getvoteaccounts
func (c *RPCClient) GetVoteAccounts(ctx context.Context, commitment Commitment) (*GetVoteAccountsResponse, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getVoteAccounts", []interface{}{commitment}))
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

	return &resp, nil
}
