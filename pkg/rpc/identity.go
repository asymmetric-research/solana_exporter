package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type GetIdentityResponse struct {
	Result struct {
		Identity string `json:"identity"`
	} `json:"result"`
	Error rpcError `json:"error"`
}

// https://docs.solana.com/developing/clients/jsonrpc-api#getidentity
func (c *RPCClient) GetIdentity(ctx context.Context) (string, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getIdentity", []interface{}{}))

	if body == nil {
		return "", fmt.Errorf("RPC call failed: Body empty")
	}

	if err != nil {
		return "", fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("identity response: %v", string(body))

	var resp GetIdentityResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return "", fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result.Identity, nil
}
