package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type getSlotResponse struct {
	Result int64 `json:"result"`
}

// https://solana.com/docs/rpc/http/getslot
func (c *Client) GetSlot(ctx context.Context) (int64, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getSlot", []interface{}{}))
	if err != nil {
		return 0, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getSlot response: %v", string(body))

	var resp getSlotResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("failed to decode response body: %w", err)
	}

	return resp.Result, nil
}
