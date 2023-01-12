package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

type (
	Balance struct {
		Context struct {
			ApiVersion string `json:"apiVersion"`
			Slot       int    `json:"slot"`
		} `json:"context"`
		Value int `json:"value"`
	}

	GetBalanceResponse struct {
		Result Balance  `json:"result"`
		Error  rpcError `json:"error"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getbalance
func (c *RPCClient) GetBalance(ctx context.Context, params []interface{}) (*GetBalanceResponse, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getBalance", params))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %v", err)
	}

	var resp GetBalanceResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}
