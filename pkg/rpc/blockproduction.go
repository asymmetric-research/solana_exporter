package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

type (
	BlockResult map[string][]int

	BlockProduction struct {
		Context struct {
			ApiVersion	string `json:"apiVersion"`
			Slot		int `json:"slot"`
		} `json:"context"`
		Value struct {
			ByIdentity BlockResult `json:"byIdentity"`
			Range struct {
				FirstSlot	int `json:"firstSlot"`
				LastSlot	int `json:"lastSlot"`
			} `json:"range"`
		} `json:"value"`
	}

	GetBlockProductionResponse struct {
		Result BlockProduction `json:"result"`
		Error rpcError `json:"error"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#getblockproduction
func (c *RPCClient) GetBlockProduction(ctx context.Context, params []interface{}) (*GetBlockProductionResponse, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getBlockProduction", params))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %v", err)
	}

	var resp GetBlockProductionResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}
