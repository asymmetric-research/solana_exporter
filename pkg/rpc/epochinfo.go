package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getepochinfo
func (c *RPCClient) GetEpochInfo(ctx context.Context, commitment Commitment) (*EpochInfo, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getEpochInfo", []interface{}{commitment}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("epoch info response: %v", string(body))

	var resp GetEpochInfoResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result, nil
}
