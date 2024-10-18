package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	"strconv"
	"time"
)

var (
	isHealthy = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solana_is_healthy",
		Help: "Is node healthy",
	})

	numSlotsBehind = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solana_num_slots_behind",
		Help: "Number of slots behind",
	})
)

func init() {
	prometheus.MustRegister(isHealthy)
	prometheus.MustRegister(numSlotsBehind)

}
func extractNumSlotsBehind(data map[string]any) (int, error) {
	if val, ok := data["NumSlotsBehind"]; ok {
		// Type assert if it's a float64 (common for numbers in JSON)
		switch v := val.(type) {
		case float64:
			return int(v), nil
		case string:
			// If it's a string, try to convert it to an int
			num, err := strconv.Atoi(v)
			if err != nil {
				return 0, fmt.Errorf("failed to convert string to int: %w", err)
			}
			return num, nil
		default:
			return 0, fmt.Errorf("unexpected type for NumSlotsBehind: %T", v)
		}
	}

	return 0, fmt.Errorf("NumSlotsBehind key not found in data")
}

func (c *SolanaCollector) WatchHealth(ctx context.Context) {
	ticker := time.NewTicker(slotPacerSchedule)

	for {
		<-ticker.C

		// Get current slot height and epoch info

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := c.rpcClient.GetHealth(ctx)
		if err != nil {
			klog.Infof("failed to fetch info info, retrying: %v", err)
			cancel()
			continue
		}
		cancel()
		isNodeHealthy := 1
		nodeNumSlotsBehind := 0
		if err != nil {
			var rpcError *rpc.RPCError
			if errors.As(err, &rpcError) {
				if rpcError.Code != 0 {
					isNodeHealthy = 0
				}
				nodeNumSlotsBehind, _ = extractNumSlotsBehind(rpcError.Data)

			}

		}
		isHealthy.Set(float64(isNodeHealthy))

		numSlotsBehind.Set(float64(nodeNumSlotsBehind))
	}
}
