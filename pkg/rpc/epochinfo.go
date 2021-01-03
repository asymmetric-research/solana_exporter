package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
)

type (
	EpochInfo struct {
		// Current absolute slot in epoch
		AbsoluteSlot int64 `json:"absoluteSlot"`
		// Current block height
		BlockHeight int64 `json:"blockHeight"`
		// Current epoch number
		Epoch int64 `json:"epoch"`
		// Current slot relative to the start of the current epoch
		SlotIndex int64 `json:"slotIndex"`
		// Number of slots in this epoch
		SlotsInEpoch int64 `json:"slotsInEpoch"`
		// Total number of transactions ever (?)
		TransactionCount int64 `json:"transactionCount"`
	}

	GetEpochInfoResponse struct {
		Result EpochInfo `json:"result"`
		Error  rpcError  `json:"error"`
	}
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
