package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/klog/v2"
	"net/http"
)

type (
	Client struct {
		httpClient http.Client
		rpcAddr    string
	}

	rpcError1 struct {
		Message string `json:"message"`
		Code    int64  `json:"id"`
	}

	rpcError2 struct { // TODO: combine these error types into a single one
		Message string `json:"message"`
		Code    int64  `json:"code"`
	}

	rpcRequest struct {
		Version string        `json:"jsonrpc"`
		ID      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	Commitment string
)

// Provider is an interface that defines the methods required to interact with the Solana blockchain.
// It provides methods to retrieve block production information, epoch info, slot info, vote accounts, and node version.
type Provider interface {

	// GetBlockProduction retrieves the block production information for the specified slot range.
	// The method takes a context for cancellation, and pointers to the first and last slots of the range.
	// It returns a BlockProduction struct containing the block production details, or an error if the operation fails.
	GetBlockProduction(ctx context.Context, firstSlot *int64, lastSlot *int64) (BlockProduction, error)

	// GetEpochInfo retrieves the information regarding the current epoch.
	// The method takes a context for cancellation and a commitment level to specify the desired state.
	// It returns a pointer to an EpochInfo struct containing the epoch details, or an error if the operation fails.
	GetEpochInfo(ctx context.Context, commitment Commitment) (*EpochInfo, error)

	// GetSlot retrieves the current slot number.
	// The method takes a context for cancellation.
	// It returns the current slot number as an int64, or an error if the operation fails.
	GetSlot(ctx context.Context) (int64, error)

	// GetVoteAccounts retrieves the vote accounts information.
	// The method takes a context for cancellation and a slice of parameters to filter the vote accounts.
	// It returns a pointer to a VoteAccounts struct containing the vote accounts details,
	// or an error if the operation fails.
	GetVoteAccounts(ctx context.Context, params []interface{}) (*VoteAccounts, error)

	// GetVersion retrieves the version of the Solana node.
	// The method takes a context for cancellation.
	// It returns a string containing the version information, or an error if the operation fails.
	GetVersion(ctx context.Context) (string, error)
}

func (c Commitment) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"commitment": string(c)})
}

const (
	// Most recent block confirmed by supermajority of the cluster as having reached maximum lockout.
	CommitmentMax Commitment = "max"
	// CommitmentRoot Most recent block having reached maximum lockout on this node.
	CommitmentRoot Commitment = "root"
	// Most recent block that has been voted on by supermajority of the cluster (optimistic confirmation).
	CommitmentSingleGossip Commitment = "singleGossip"
	// The node will query its most recent block. Note that the block may not be complete.
	CommitmentRecent Commitment = "recent"
)

func NewRPCClient(rpcAddr string) *Client {
	c := &Client{
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

func (c *Client) rpcRequest(ctx context.Context, data io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcAddr, data)
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) getResponse(ctx context.Context, method string, params []interface{}, result HasRPCError) error {
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

func (c *Client) GetEpochInfo(ctx context.Context, commitment Commitment) (*EpochInfo, error) {
	var resp response[EpochInfo]
	if err := c.getResponse(ctx, "getEpochInfo", []interface{}{commitment}, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

func (c *Client) GetVoteAccounts(ctx context.Context, params []interface{}) (*VoteAccounts, error) {
	var resp response[VoteAccounts]
	if err := c.getResponse(ctx, "getVoteAccounts", params, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

func (c *Client) GetVersion(ctx context.Context) (string, error) {
	var resp response[VersionInfo]
	if err := c.getResponse(ctx, "getVersion", []interface{}{}, &resp); err != nil {
		return "", err
	}
	return resp.Result.Version, nil
}

func (c *Client) GetSlot(ctx context.Context) (int64, error) {
	var resp response[int64]
	if err := c.getResponse(ctx, "getSlot", []interface{}{}, &resp); err != nil {
		return 0, err
	}
	return resp.Result, nil
}
