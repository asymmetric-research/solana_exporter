package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
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

func (c *solanaCollector) WatchHealth(ctx context.Context) {
	ticker := time.NewTicker(slotPacerSchedule)

	for {
		<-ticker.C

		// Get current slot height and epoch info

		ctx, cancel := context.WithTimeout(ctx, httpTimeout)
		healthResult, healthError, err := c.rpcClient.GetHealth(ctx)
		if err != nil {
			klog.Infof("failed to fetch info info, retrying: %v", err)
			cancel()
			continue
		}
		cancel()

		isNodeHealthy := 0
		if healthResult != nil {
			isNodeHealthy = 1
		}
		isHealthy.Set(float64(isNodeHealthy))
		var nodeNumSlotsBehind int64
		if healthError != nil {
			nodeNumSlotsBehind = healthError.Data.NumSlotsBehind
		}

		numSlotsBehind.Set(float64(nodeNumSlotsBehind))

	}
}
