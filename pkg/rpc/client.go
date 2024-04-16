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

func (c *RPCClient) rpcCall(ctx context.Context, method string, params []interface{}, result HasRPCError) error {
	body, err := c.rpcRequest(ctx, formatRPCRequest(method, params))
	// check if there was an error making the request:
	if err != nil {
		return fmt.Errorf("%s RPC call failed: %w", method, err)
	}
	// log response:
	klog.V(2).Infof("%s response: %v", method, string(body))

	// unmarshal the response into the predicted format
	if err = json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to decode %s response body: %w", method, err)
	}

	if result.getError().Code != 0 {
		return fmt.Errorf("RPC error: %d %v", result.getError().Code, result.getError().Message)
	}

	return nil
}

func (c *RPCClient) GetConfirmedBlocks(ctx context.Context, startSlot, endSlot int64) ([]int64, error) {
	var resp Response[[]int64]
	if err := c.rpcCall(ctx, "getConfirmedBlocks", []interface{}{startSlot, endSlot}, &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *RPCClient) GetEpochInfo(ctx context.Context, commitment Commitment) (*EpochInfo, error) {
	var resp Response[EpochInfo]
	if err := c.rpcCall(ctx, "getEpochInfo", []interface{}{commitment}, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

func (c *RPCClient) GetLeaderSchedule(ctx context.Context, epochSlot int64) (LeaderSchedule, error) {
	var resp Response[LeaderSchedule]
	if err := c.rpcCall(ctx, "getLeaderSchedule", []interface{}{epochSlot}, &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *RPCClient) GetVoteAccounts(ctx context.Context, params []interface{}) (*VoteAccounts, error) {
	var resp Response[VoteAccounts]
	if err := c.rpcCall(ctx, "getVoteAccounts", params, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

func (c *RPCClient) GetVersion(ctx context.Context) (*string, error) {
	var resp Response[VersionInfo]
	if err := c.rpcCall(ctx, "getVersion", []interface{}{}, &resp); err != nil {
		return nil, err
	}
	return &resp.Result.Version, nil
}
