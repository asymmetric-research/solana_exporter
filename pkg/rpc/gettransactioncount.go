package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type (
	GetTransactionCountResponse struct {
		Result int64    `json:"result"`
		Error  rpcError `json:"error"`
	}
)

// GetTransactionCount is https://docs.solana.com/developing/clients/jsonrpc-api#gettransactioncount
func (c *RPCClient) GetTransactionCount(ctx context.Context, commitment Commitment) (*int64, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getTransactionCount", []interface{}{commitment}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getTransactionCount  response: %v", string(body))

	var resp GetTransactionCountResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result, nil
}
