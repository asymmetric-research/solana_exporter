package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var httpClient http.Client

type (
	VoteAccount struct {
		ActivatedStake   int64   `json:"activatedStake"`
		Commission       int     `json:"commission"`
		EpochCredits     [][]int `json:"epochCredits"`
		EpochVoteAccount bool    `json:"epochVoteAccount"`
		LastVote         int     `json:"lastVote"`
		NodePubkey       string  `json:"nodePubkey"`
		RootSlot         int     `json:"rootSlot"`
		VotePubkey       string  `json:"votePubkey"`
	}

	GetVoteAccountsResponse struct {
		Result struct {
			Current    []VoteAccount `json:"current"`
			Delinquent []VoteAccount `json:"delinquent"`
		} `json:"result"`
	}
)

func GetVoteAccounts(ctx context.Context, rpcAddr string) (*GetVoteAccountsResponse, error) {
	var (
		voteAccounts GetVoteAccountsResponse
		body         []byte
		err          error
	)

	req, err := http.NewRequestWithContext(ctx, "POST", rpcAddr,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":1, "method":"getVoteAccounts", "params":[{"commitment":"recent"}]}`))
	if err != nil {
		panic(err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RPC call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err = json.Unmarshal(body, &voteAccounts); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return &voteAccounts, nil
}
