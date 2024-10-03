package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/klog/v2"
)

type (
	errorData struct {
		NumSlotsBehind int64 `json:"numSlotsBehind"`
	}

	GetHealthRpcError struct {
		Message string    `json:"message"`
		Data    errorData `json:"data"`
		Code    int64     `json:"code"`
	}

	getHealthResponse struct {
		Result string            `json:"result"`
		Error  GetHealthRpcError `json:"error"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#gethealth
func (c *Client) GetHealth(ctx context.Context) (*string, *GetHealthRpcError, error) {
	// only retrieve results of method

	body, err := c.rpcRequest(ctx, formatRPCRequest("getHealth", []interface{}{}))
	if err != nil {
		return nil, nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getHealth response: %v", string(body))

	var resp getHealthResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, &resp.Error, nil
	}

	return &resp.Result, nil, nil
}
