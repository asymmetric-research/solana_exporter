package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
)

type (
	LeaderSchedule map[string][]int64

	GetLeaderScheduleResponse struct {
		Result LeaderSchedule `json:"result"`
		Error  rpcError       `json:"error"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getleaderschedule
func (c *RPCClient) GetLeaderSchedule(ctx context.Context, epochSlot int64) (LeaderSchedule, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getLeaderSchedule", []interface{}{epochSlot}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(3).Infof("getLeaderSchedule response: %v", string(body))

	var resp GetLeaderScheduleResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}
