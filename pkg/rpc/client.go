package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
)

type (
	RPCClient struct {
		httpClient http.Client
		rpcAddr    string
	}

	rpcError struct {
		Message string `json:"message"`
		Code    int64  `json:"id"`
	}

	rpcRequest struct {
		Version string        `json:"jsonrpc"`
		ID      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	Commitment string
)

func (c Commitment) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"commitment": string(c)})
}

const (
	// Most recent block confirmed by supermajority of the cluster as having reached maximum lockout.
	CommitmentMax Commitment = "max"
	// Most recent block having reached maximum lockout on this node.
	CommitmentRoot Commitment = "root"
	// Most recent block that has been voted on by supermajority of the cluster (optimistic confirmation).
	CommitmentSingleGossip Commitment = "singleGossip"
	// The node will query its most recent block. Note that the block may not be complete.
	CommitmentRecent Commitment = "recent"
)

func NewRPCClient(rpcAddr string) *RPCClient {
	c := &RPCClient{
		httpClient: http.Client{},
		rpcAddr:    rpcAddr,
	}

	return c
}

func formatRPCRequest(method string, params []interface{}) io.Reader {
	r := &rpcRequest{
		Version: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	b, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}

	klog.V(2).Infof("jsonrpc request: %s", string(b))
	return bytes.NewBuffer(b)
}

func (c *RPCClient) rpcRequest(ctx context.Context, data io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcAddr, data)
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// https://docs.solana.com/developing/clients/jsonrpc-api#getblocktime
func (c *RPCClient) GetBlockTime(ctx context.Context, slot int64) (int64, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getBlockTime", []interface{}{slot}))
	if err != nil {
		return 0, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getBlockTime response: %v", string(body))

	var resp GetBlockTimeResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return 0, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

// https://docs.solana.com/developing/clients/jsonrpc-api#getconfirmedblocks
func (c *RPCClient) GetConfirmedBlocks(ctx context.Context, startSlot, endSlot int64) ([]int64, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getConfirmedBlocks", []interface{}{startSlot, endSlot}))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getBlockTime response: %v", string(body))

	var resp GetConfirmedBlocksResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return resp.Result, nil
}

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

// https://docs.solana.com/developing/clients/jsonrpc-api#getvoteaccounts
func (c *RPCClient) GetVoteAccounts(ctx context.Context, params []interface{}) (*GetVoteAccountsResponse, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getVoteAccounts", params))
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(3).Infof("getVoteAccounts response: %v", string(body))

	var resp GetVoteAccountsResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}

func (c *RPCClient) GetVersion(ctx context.Context) (*string, error) {
	body, err := c.rpcRequest(ctx, formatRPCRequest("getVersion", []interface{}{}))

	if body == nil {
		return nil, fmt.Errorf("RPC call failed: Body empty")
	}

	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("version response: %v", string(body))

	var resp GetVersionResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	return &resp.Result.Version, nil
}
