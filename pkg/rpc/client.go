package rpc

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
)

type RpcClient struct {
	httpClient http.Client
	rpcAddr    string
}

func NewRPCClient(rpcAddr string) *RpcClient {
	c := &RpcClient{
		httpClient: http.Client{},
		rpcAddr:    rpcAddr,
	}

	return c
}

func (c *RpcClient) rpcRequest(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcAddr, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}
