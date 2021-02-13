package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type GetRecentBlockhashResult struct {
	Context struct {
		Slot int `json:"slot"`
	} `json:"context"`
	Value struct {
		Blockhash     string `json:"blockhash"`
		FeeCalculator struct {
			LamportsPerSignature int `json:"lamportsPerSignature"`
		} `json:"feeCalculator"`
	} `json:"value"`
}

type (
	GetRecentBlockhashResponse struct {
		Result GetRecentBlockhashResult `json:"result"`
		Error  rpcError                 `json:"error"`
	}
)

// GetRecentBlockhash is https://docs.solana.com/developing/clients/jsonrpc-api#getrecentblockhash
func (c *RPCClient) GetRecentBlockhash(ctx context.Context, commitment Commitment) (*GetRecentBlockhashResult, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getRecentBlockhash", []interface{}{commitment}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getRecentBlockhash  response: %v", string(body))

	var resp GetRecentBlockhashResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result, nil
}
