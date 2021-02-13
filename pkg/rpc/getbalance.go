package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type GetBalanceResult struct {
	Context struct {
		Slot int `json:"slot"`
	} `json:"context"`
	Value int64 `json:"value"`
}

type (
	GetBalanceResponse struct {
		Result GetBalanceResult `json:"result"`
		Error  rpcError         `json:"error"`
	}
)

// GetBalance is https://docs.solana.com/developing/clients/jsonrpc-api#getbalance
func (c *RPCClient) GetBalance(ctx context.Context, params string) (*GetBalanceResult, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getBalance", []interface{}{params}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getBalance  response: %v", string(body))

	var resp GetBalanceResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result, nil
}
