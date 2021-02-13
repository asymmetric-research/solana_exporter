package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type GetClusterNodesResult struct {
	FeatureSet int64  `json:"featureSet"`
	Gossip     string `json:"gossip"`
	Pubkey     string `json:"pubkey"`
	RPC        string `json:"rpc"`
	Tpu        string `json:"tpu"`
	Version    string `json:"version"`
}

type (
	GetClusterNodesResponse struct {
		Result []GetClusterNodesResult `json:"result"`
		Error  rpcError                `json:"error"`
	}
)

// GetClusterNodes is https://docs.solana.com/developing/clients/jsonrpc-api#getclusternodes
func (c *RPCClient) GetClusterNodes(ctx context.Context) ([]GetClusterNodesResult, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getClusterNodes", []interface{}{}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getClusterNodes  response: %v", string(body))

	var resp GetClusterNodesResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}
