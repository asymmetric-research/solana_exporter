package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
)

type (
	GetBlockTimeResponse struct {
		Result int64    `json:"result"`
		Error  rpcError `json:"error"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getblocktime
func (c *RPCClient) GetBlockTime(ctx context.Context, slot int64) (int64, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getBlockTime", []interface{}{slot}))
	if err != nil {
		return 0, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getBlockTime response: %v", string(body))

	var resp GetBlockTimeResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return 0, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}
