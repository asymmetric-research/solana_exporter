package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getvoteaccounts
func (c *RPCClient) GetVoteAccounts(ctx context.Context, params []interface{}) (*GetVoteAccountsResponse, error) {
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

	return &resp, nil
}
