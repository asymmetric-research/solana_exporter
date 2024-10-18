package rpc

import (
	"context"
)

type (
	ErrorData struct {
		NumSlotsBehind int64 `json:"numSlotsBehind"`
	}

	GetHealthRpcError struct {
		Message string    `json:"message"`
		Data    ErrorData `json:"data"`
		Code    int64     `json:"code"`
	}

	getHealthResponse struct {
		jsonrpc string
		Result  string   `json:"result"`
		Error   RPCError `json:"error"`
		Id      int      `json:"id"`
	}
)

// https://docs.solana.com/developing/clients/jsonrpc-api#gethealth
func (c *Client) GetHealth(ctx context.Context) (*string, error) {
	var resp response[string]

	if err := getResponse(ctx, c, "getHealth", []any{}, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}
