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

	rpcError struct {
		Message string `json:"message"`
		Code    int64  `json:"code"`
	}

	rpcRequest struct {
		Version string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
	}

	Commitment string
)

// Provider is an interface that defines the methods required to interact with the Solana blockchain.
// It provides methods to retrieve block production information, epoch info, slot info, vote accounts, and node version.
type Provider interface {

	// GetBlockProduction retrieves the block production information for the specified slot range.
	// The method takes a context for cancellation, and pointers to the first and last slots of the range.
	// It returns a BlockProduction struct containing the block production details, or an error if the operation fails.
	GetBlockProduction(
		ctx context.Context, identity *string, firstSlot *int64, lastSlot *int64,
	) (*BlockProduction, error)

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
	GetVoteAccounts(ctx context.Context, commitment Commitment, votePubkey *string) (*VoteAccounts, error)

	// GetVersion retrieves the version of the Solana node.
	// The method takes a context for cancellation.
	// It returns a string containing the version information, or an error if the operation fails.
	GetVersion(ctx context.Context) (string, error)

	// GetBalance returns the SOL balance of the account at the provided address
	GetBalance(ctx context.Context, address string) (float64, error)

	// GetInflationReward returns the inflation rewards (in lamports) awarded to the given addresses (vote accounts)
	// during the given epoch.
	GetInflationReward(
		ctx context.Context, addresses []string, commitment Commitment, epoch *int64, minContextSlot *int64,
	) ([]InflationReward, error)
}

func (c Commitment) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{"commitment": string(c)})
}

const (
	// LamportsInSol is the number of lamports in 1 SOL (a billion)
	LamportsInSol = 1_000_000_000
	// CommitmentFinalized level offers the highest level of certainty for a transaction on the Solana blockchain.
	// A transaction is considered “Finalized” when it is included in a block that has been confirmed by a
	// supermajority of the stake, and at least 31 additional confirmed blocks have been built on top of it.
	CommitmentFinalized Commitment = "finalized"
	// CommitmentConfirmed level is reached when a transaction is included in a block that has been voted on
	// by a supermajority (66%+) of the network’s stake.
	CommitmentConfirmed Commitment = "confirmed"
	// CommitmentProcessed level represents a transaction that has been received by the network and included in a block.
	CommitmentProcessed Commitment = "processed"
)

func NewRPCClient(rpcAddr string) *Client {
	return &Client{httpClient: http.Client{}, rpcAddr: rpcAddr}
}

func (c *Client) getResponse(ctx context.Context, method string, params []any, result HasRPCError) error {
	// format request:
	request := &rpcRequest{Version: "2.0", ID: 1, Method: method, Params: params}
	buffer, err := json.Marshal(request)
	if err != nil {
		klog.Fatalf("failed to marshal request: %v", err)
	}
	klog.V(2).Infof("jsonrpc request: %s", string(buffer))

	// make request:
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcAddr, bytes.NewBuffer(buffer))
	if err != nil {
		klog.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s RPC call failed: %w", method, err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error processing %s rpc call: %w", method, err)
	}
	// log response:
	klog.V(2).Infof("%s response: %v", method, string(body))

	// unmarshal the response into the predicted format
	if err = json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to decode %s response body: %w", method, err)
	}

	// last error check:
	if result.getError().Code != 0 {
		return fmt.Errorf("RPC error: %d %v", result.getError().Code, result.getError().Message)
	}

	return nil
}

// GetEpochInfo returns information about the current epoch.
// See API docs: https://solana.com/docs/rpc/http/getepochinfo
func (c *Client) GetEpochInfo(ctx context.Context, commitment Commitment) (*EpochInfo, error) {
	var resp response[EpochInfo]
	if err := c.getResponse(ctx, "getEpochInfo", []any{commitment}, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// GetVoteAccounts returns the account info and associated stake for all the voting accounts in the current bank.
// See API docs: https://solana.com/docs/rpc/http/getvoteaccounts
func (c *Client) GetVoteAccounts(
	ctx context.Context, commitment Commitment, votePubkey *string,
) (*VoteAccounts, error) {
	// format params:
	config := map[string]string{"commitment": string(commitment)}
	if votePubkey != nil {
		config["votePubkey"] = *votePubkey
	}

	var resp response[VoteAccounts]
	if err := c.getResponse(ctx, "getVoteAccounts", []any{config}, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// GetVersion returns the current Solana version running on the node.
// See API docs: https://solana.com/docs/rpc/http/getversion
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	var resp response[struct {
		Version string `json:"solana-core"`
	}]
	if err := c.getResponse(ctx, "getVersion", []any{}, &resp); err != nil {
		return "", err
	}
	return resp.Result.Version, nil
}

// GetSlot returns the slot that has reached the given or default commitment level.
// See API docs: https://solana.com/docs/rpc/http/getslot
func (c *Client) GetSlot(ctx context.Context, commitment Commitment) (int64, error) {
	config := map[string]string{"commitment": string(commitment)}
	var resp response[int64]
	if err := c.getResponse(ctx, "getSlot", []any{config}, &resp); err != nil {
		return 0, err
	}
	return resp.Result, nil
}

// GetBlockProduction returns recent block production information from the current or previous epoch.
// See API docs: https://solana.com/docs/rpc/http/getblockproduction
func (c *Client) GetBlockProduction(
	ctx context.Context, identity *string, firstSlot *int64, lastSlot *int64,
) (*BlockProduction, error) {
	// can't provide a last slot without a first:
	if firstSlot == nil && lastSlot != nil {
		klog.Fatalf("can't provide a last slot without a first!")
	}

	// format params:
	config := make(map[string]any)
	if identity != nil {
		config["identity"] = *identity
	}
	if firstSlot != nil {
		blockRange := map[string]int64{"firstSlot": *firstSlot}
		if lastSlot != nil {
			// make sure first and last slot are in order:
			if *firstSlot > *lastSlot {
				err := fmt.Errorf("last slot %v is greater than first slot %v", *lastSlot, *firstSlot)
				klog.Fatalf("%v", err)
			}
			blockRange["lastSlot"] = *lastSlot
		}
		config["range"] = blockRange
	}

	var params []any
	if len(config) > 0 {
		params = append(params, config)
	}

	// make request:
	var resp response[contextualResult[BlockProduction]]
	if err := c.getResponse(ctx, "getBlockProduction", params, &resp); err != nil {
		return nil, err
	}
	return &resp.Result.Value, nil
}

// GetBalance returns the lamport balance of the account of provided pubkey.
// See API docs:https://solana.com/docs/rpc/http/getbalance
func (c *Client) GetBalance(ctx context.Context, address string) (float64, error) {
	var resp response[contextualResult[int64]]
	if err := c.getResponse(ctx, "getBalance", []any{address}, &resp); err != nil {
		return 0, err
	}
	return float64(resp.Result.Value) / float64(LamportsInSol), nil
}

// GetInflationReward returns the inflation / staking reward for a list of addresses for an epoch.
// See API docs: https://solana.com/docs/rpc/http/getinflationreward
func (c *Client) GetInflationReward(
	ctx context.Context, addresses []string, commitment Commitment, epoch *int64, minContextSlot *int64,
) ([]InflationReward, error) {
	// format params:
	config := map[string]any{"commitment": string(commitment)}
	if epoch != nil {
		config["epoch"] = *epoch
	}
	if minContextSlot != nil {
		config["minContextSlot"] = *minContextSlot
	}

	var resp response[[]InflationReward]
	if err := c.getResponse(ctx, "getInflationReward", []any{addresses, config}, &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// GetLeaderSchedule returns the leader schedule for an epoch.
// See API docs: https://solana.com/docs/rpc/http/getleaderschedule
func (c *Client) GetLeaderSchedule(ctx context.Context, commitment Commitment) (map[string][]int64, error) {
	config := map[string]any{"commitment": string(commitment)}
	var resp response[map[string][]int64]
	if err := c.getResponse(ctx, "getLeaderSchedule", []any{nil, config}, &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

// GetBlock returns identity and transaction information about a confirmed block in the ledger.
// See API docs: https://solana.com/docs/rpc/http/getblock
func (c *Client) GetBlock(ctx context.Context, slot int64, commitment Commitment) (*Block, error) {
	if commitment == CommitmentProcessed {
		klog.Fatalf("commitment %v is not supported for GetBlock", commitment)
	}
	config := map[string]any{
		"commitment":         commitment,
		"encoding":           "json", // this is default, but no harm in specifying it
		"transactionDetails": "none", // for now, can hard-code this out, as we don't need it
		"rewards":            true,   // what we here for!
	}
	var resp response[Block]
	if err := c.getResponse(ctx, "getBlock", []any{slot, config}, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}
