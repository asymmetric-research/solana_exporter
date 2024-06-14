package main

import (
	"context"
	"github.com/certusone/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSolanaCollector_WatchSlots_Local(t *testing.T) {
	// using short to skip this test while I figure out how build tags work
	if testing.Short() {
		t.Skip("skipping local test in short mode.")
	}
	// reset metrics before running tests:
	leaderSlotsTotal.Reset()
	leaderSlotsByEpoch.Reset()

	// create clients:
	client := rpc.NewRPCClient("http://localhost:8899")
	collector := createSolanaCollector(
		client,
		slotPacerSchedule,
	)
	prometheus.NewPedanticRegistry().MustRegister(collector)

	// start client/collector and wait a bit:
	ctx, cancel := context.WithCancel(context.Background())
	go collector.WatchSlots(ctx)
	time.Sleep(time.Second)

	initial := getSlotMetricValues()

	for i := 0; i < 10; i++ {
		// wait a bit then get new metrics
		time.Sleep(2 * time.Second)
		final := getSlotMetricValues()

		// make sure things are changing correctly:
		assertSlotMetricsChangeCorrectly(t, initial, final)

		// some sense checks against time of creation:
		assert.Greater(t, final.EpochNumber, float64(628))
		assert.Greater(t, final.SlotHeight, float64(271_760_309))
		assert.Greater(t, final.TotalTransactions, float64(295_561_750_573))

		// other sense checks:
		assert.Equal(t, float64(432_000), final.EpochLastSlot-final.EpochFirstSlot)

		// make current final the new initial (for next iteration)
		initial = final
	}

	// cancel and wait for cancellation:
	cancel()
	time.Sleep(time.Second)
}
