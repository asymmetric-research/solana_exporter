package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
)

type (
	GetConfirmedBlocksResponse struct {
		Result []int64  `json:"result"`
		Error  rpcError `json:"error"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getconfirmedblocks
func (c *RPCClient) GetConfirmedBlocks(ctx context.Context, startSlot, endSlot int64) ([]int64, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getConfirmedBlocks", []interface{}{startSlot, endSlot}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getBlockTime response: %v", string(body))

	var resp GetConfirmedBlocksResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}
