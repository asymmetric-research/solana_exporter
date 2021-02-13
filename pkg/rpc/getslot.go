package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type (
	GetSlotResponse struct {
		Result int64    `json:"result"`
		Error  rpcError `json:"error"`
	}
)

// GetSlot is https://docs.solana.com/developing/clients/jsonrpc-api#GetSlot
func (c *RPCClient) GetSlot(ctx context.Context) (*int64, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getSlot", []interface{}{}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("GetSlot  response: %v", string(body))

	var resp GetSlotResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result, nil
}
