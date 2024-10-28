package main

import (
	"context"
	"fmt"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type slotMetricValues struct {
	SlotHeight        float64
	TotalTransactions float64
	EpochNumber       float64
	EpochFirstSlot    float64
	EpochLastSlot     float64
}

func getSlotMetricValues(watcher *SlotWatcher) slotMetricValues {
	return slotMetricValues{
		SlotHeight:        testutil.ToFloat64(watcher.SlotHeightMetric),
		TotalTransactions: testutil.ToFloat64(watcher.TotalTransactionsMetric),
		EpochNumber:       testutil.ToFloat64(watcher.EpochNumberMetric),
		EpochFirstSlot:    testutil.ToFloat64(watcher.EpochFirstSlotMetric),
		EpochLastSlot:     testutil.ToFloat64(watcher.EpochLastSlotMetric),
	}
}

func assertSlotMetricsChangeCorrectly(t *testing.T, initial slotMetricValues, final slotMetricValues) {
	// make sure that things have increased
	assert.Greaterf(
		t,
		final.SlotHeight,
		initial.SlotHeight,
		"Slot has not increased! (%v -> %v)",
		initial.SlotHeight,
		final.SlotHeight,
	)
	assert.Greaterf(
		t,
		final.TotalTransactions,
		initial.TotalTransactions,
		"Total transactions have not increased! (%v -> %v)",
		initial.TotalTransactions,
		final.TotalTransactions,
	)
	assert.GreaterOrEqualf(
		t,
		final.EpochNumber,
		initial.EpochNumber,
		"Epoch number has decreased! (%v -> %v)",
		initial.EpochNumber,
		final.EpochNumber,
	)
}

func TestSlotWatcher_WatchSlots_Static(t *testing.T) {
	ctx := context.Background()

	config := newTestConfig(true)

	_, client := NewDynamicRpcClient(t, 35)

	watcher := NewSlotWatcher(client, config)
	// reset metrics before running tests:
	watcher.LeaderSlotsMetric.Reset()
	watcher.LeaderSlotsByEpochMetric.Reset()

	go watcher.WatchSlots(ctx)

	// make sure inflation rewards are collected:
	epochInfo, err := client.GetEpochInfo(ctx, rpc.CommitmentFinalized)
	assert.NoError(t, err)
	err = watcher.fetchAndEmitInflationRewards(ctx, epochInfo.Epoch)
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)

	type testCase struct {
		expectedValue float64
		metric        prometheus.Gauge
	}

	// epoch info tests:
	firstSlot, lastSlot := GetEpochBounds(epochInfo)
	tests := []testCase{
		{expectedValue: float64(epochInfo.AbsoluteSlot), metric: watcher.SlotHeightMetric},
		{expectedValue: float64(epochInfo.TransactionCount), metric: watcher.TotalTransactionsMetric},
		{expectedValue: float64(epochInfo.Epoch), metric: watcher.EpochNumberMetric},
		{expectedValue: float64(firstSlot), metric: watcher.EpochFirstSlotMetric},
		{expectedValue: float64(lastSlot), metric: watcher.EpochLastSlotMetric},
	}

	// add inflation reward tests:
	inflationRewards, err := client.GetInflationReward(ctx, rpc.CommitmentFinalized, votekeys, 2)
	assert.NoError(t, err)
	for i, rewardInfo := range inflationRewards {
		epoch := fmt.Sprintf("%v", epochInfo.Epoch)
		tests = append(
			tests,
			testCase{
				expectedValue: float64(rewardInfo.Amount) / float64(rpc.LamportsInSol),
				metric:        watcher.InflationRewardsMetric.WithLabelValues(votekeys[i], epoch),
			},
		)
	}

	for _, testCase := range tests {
		name := extractName(testCase.metric.Desc())
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedValue, testutil.ToFloat64(testCase.metric))
		})
	}
}

func TestSlotWatcher_WatchSlots_Dynamic(t *testing.T) {
	// create clients:
	server, client := NewDynamicRpcClient(t, 35)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := newTestConfig(true)
	collector := NewSolanaCollector(client, config)
	watcher := NewSlotWatcher(client, config)
	// reset metrics before running tests:
	watcher.LeaderSlotsMetric.Reset()
	watcher.LeaderSlotsByEpochMetric.Reset()
	prometheus.NewPedanticRegistry().MustRegister(collector)

	// start client/collector and wait a bit:

	go server.Run(ctx)
	time.Sleep(time.Second)

	go watcher.WatchSlots(ctx)
	time.Sleep(time.Second)

	initial := getSlotMetricValues(watcher)

	// wait a bit:
	var epochChanged bool
	for i := 0; i < 5; i++ {
		// wait a bit then get new metrics
		time.Sleep(time.Second)
		final := getSlotMetricValues(watcher)

		// make sure things are changing correctly:
		assertSlotMetricsChangeCorrectly(t, initial, final)

		// sense check to make sure the exporter is not "ahead" of the client (due to double counting or whatever)
		assert.LessOrEqualf(
			t,
			int(final.SlotHeight),
			server.Slot,
			"Exporter slot (%v) ahead of client slot (%v)!",
			int(final.SlotHeight),
			server.Slot,
		)
		assert.LessOrEqualf(
			t,
			int(final.TotalTransactions),
			server.TransactionCount,
			"Exporter transaction count (%v) ahead of client transaction count (%v)!",
			int(final.TotalTransactions),
			server.TransactionCount,
		)
		assert.LessOrEqualf(
			t,
			int(final.EpochNumber),
			server.Epoch,
			"Exporter epoch (%v) ahead of client epoch (%v)!",
			int(final.EpochNumber),
			server.Epoch,
		)

		// check if epoch changed
		if final.EpochNumber > initial.EpochNumber {
			epochChanged = true
		}

		// make current final the new initial (for next iteration)
		initial = final
	}

	// epoch should have changed somewhere
	assert.Truef(t, epochChanged, "Epoch has not changed!")
}
