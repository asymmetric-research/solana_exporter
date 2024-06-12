package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/klog/v2"
)

type (
	getBlockProductionConfig struct {
		Range getBlockProductionRange `json:"range,omitempty"`
	}

	getBlockProductionRange struct {
		FirstSlot int64  `json:"firstSlot"`
		LastSlot  *int64 `json:"lastSlot,omitempty"`
	}

	getBlockProductionValue struct {
		ByIdentity map[string][]int64      `json:"byIdentity"`
		Range      getBlockProductionRange `json:"range"`
	}

	getBlockProductionResult struct {
		Value getBlockProductionValue `json:"value"`
	}

	getBlockProductionResponse struct {
		Result getBlockProductionResult `json:"result"`
		Error  rpcError2                `json:"error"`
	}

	BlockProductionPerHost struct {
		LeaderSlots    int64
		BlocksProduced int64
	}

	BlockProduction struct {
		FirstSlot int64
		LastSlot  int64
		Hosts     map[string]BlockProductionPerHost
	}
)

// https://solana.com/docs/rpc/http/getblockproduction
func (c *Client) GetBlockProduction(ctx context.Context, firstSlot *int64, lastSlot *int64) (BlockProduction, error) {
	config := make([]interface{}, 0, 1)
	if firstSlot != nil {
		config = append(config,
			getBlockProductionConfig{
				Range: getBlockProductionRange{
					FirstSlot: *firstSlot,
					LastSlot:  lastSlot,
				},
			})
	}

	ret := BlockProduction{
		FirstSlot: 0,
		LastSlot:  0,
		Hosts:     nil,
	}

	body, err := c.rpcRequest(ctx, formatRPCRequest("getBlockProduction", config))
	if err != nil {
		return ret, fmt.Errorf("RPC call failed: %w", err)
	}

	klog.V(2).Infof("getBlockProduction response: %v", string(body))

	var resp getBlockProductionResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return ret, fmt.Errorf("failed to decode response body: %w", err)
	}

	if resp.Error.Code != 0 {
		return ret, fmt.Errorf("RPC error: %d %v", resp.Error.Code, resp.Error.Message)
	}

	ret.FirstSlot = resp.Result.Value.Range.FirstSlot
	ret.LastSlot = *resp.Result.Value.Range.LastSlot
	ret.Hosts = make(map[string]BlockProductionPerHost)

	for id, arr := range resp.Result.Value.ByIdentity {
		ret.Hosts[id] = BlockProductionPerHost{
			LeaderSlots:    arr[0],
			BlocksProduced: arr[1],
		}
	}

	return ret, nil
}
