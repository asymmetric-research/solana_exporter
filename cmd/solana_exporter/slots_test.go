package main

import (
	"context"
	"fmt"
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

func getSlotMetricValues() slotMetricValues {
	return slotMetricValues{
		SlotHeight:        testutil.ToFloat64(confirmedSlotHeight),
		TotalTransactions: testutil.ToFloat64(totalTransactionsTotal),
		EpochNumber:       testutil.ToFloat64(currentEpochNumber),
		EpochFirstSlot:    testutil.ToFloat64(epochFirstSlot),
		EpochLastSlot:     testutil.ToFloat64(epochLastSlot),
	}
}

func testBlockProductionMetric(
	t *testing.T,
	metric *prometheus.CounterVec,
	host string,
	status string,
) {
	hostInfo := staticBlockProduction.ByIdentity[host]
	// get expected value depending on status:
	var expectedValue float64
	switch status {
	case "valid":
		expectedValue = float64(hostInfo.BlocksProduced)
	case "skipped":
		expectedValue = float64(hostInfo.LeaderSlots - hostInfo.BlocksProduced)
	}
	// get labels (leaderSlotsByEpoch requires an extra one)
	labels := []string{status, host}
	if metric == leaderSlotsByEpoch {
		labels = append(labels, fmt.Sprintf("%d", staticEpochInfo.Epoch))
	}
	// now we can do the assertion:
	assert.Equalf(
		t,
		expectedValue,
		testutil.ToFloat64(metric.WithLabelValues(labels...)),
		"wrong value for block-production metric with labels: %s",
		labels,
	)
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

func TestSolanaCollector_WatchSlots_Static(t *testing.T) {
	// reset metrics before running tests:
	leaderSlotsTotal.Reset()
	leaderSlotsByEpoch.Reset()

	collector := createSolanaCollector(&staticRPCClient{}, 100*time.Millisecond, identities, []string{})
	watcher := SlotWatcher{client: collector.rpcClient}
	prometheus.NewPedanticRegistry().MustRegister(collector)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watcher.WatchSlots(ctx, collector.slotPace)
	time.Sleep(1 * time.Second)

	firstSlot, lastSlot := getEpochBounds(&staticEpochInfo)
	tests := []struct {
		expectedValue float64
		metric        prometheus.Gauge
	}{
		{expectedValue: float64(staticEpochInfo.AbsoluteSlot), metric: confirmedSlotHeight},
		{expectedValue: float64(staticEpochInfo.TransactionCount), metric: totalTransactionsTotal},
		{expectedValue: float64(staticEpochInfo.Epoch), metric: currentEpochNumber},
		{expectedValue: float64(firstSlot), metric: epochFirstSlot},
		{expectedValue: float64(lastSlot), metric: epochLastSlot},
	}

	for _, testCase := range tests {
		name := extractName(testCase.metric.Desc())
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedValue, testutil.ToFloat64(testCase.metric))
		})
	}

	metrics := map[string]*prometheus.CounterVec{
		"solana_leader_slots_total":    leaderSlotsTotal,
		"solana_leader_slots_by_epoch": leaderSlotsByEpoch,
	}
	statuses := []string{"valid", "skipped"}
	for name, metric := range metrics {
		// subtest for each metric:
		t.Run(name, func(t *testing.T) {
			for _, status := range statuses {
				// sub subtest for each status (as each one requires a different calc)
				t.Run(status, func(t *testing.T) {
					for _, identity := range identities {
						testBlockProductionMetric(t, metric, identity, status)
					}
				})
			}
		})
	}
}

func TestSolanaCollector_WatchSlots_Dynamic(t *testing.T) {
	// reset metrics before running tests:
	leaderSlotsTotal.Reset()
	leaderSlotsByEpoch.Reset()

	// create clients:
	client := newDynamicRPCClient()
	collector := createSolanaCollector(client, 300*time.Millisecond, identities, []string{})
	watcher := SlotWatcher{client: client}
	prometheus.NewPedanticRegistry().MustRegister(collector)

	// start client/collector and wait a bit:
	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	go client.Run(runCtx)
	time.Sleep(time.Second)

	slotsCtx, slotsCancel := context.WithCancel(context.Background())
	defer slotsCancel()
	go watcher.WatchSlots(slotsCtx, collector.slotPace)
	time.Sleep(time.Second)

	initial := getSlotMetricValues()

	// wait a bit:
	var epochChanged bool
	for i := 0; i < 5; i++ {
		// wait a bit then get new metrics
		time.Sleep(time.Second)
		final := getSlotMetricValues()

		// make sure things are changing correctly:
		assertSlotMetricsChangeCorrectly(t, initial, final)

		// sense check to make sure the exporter is not "ahead" of the client (due to double counting or whatever)
		assert.LessOrEqualf(
			t,
			int(final.SlotHeight),
			client.Slot,
			"Exporter slot (%v) ahead of client slot (%v)!",
			int(final.SlotHeight),
			client.Slot,
		)
		assert.LessOrEqualf(
			t,
			int(final.TotalTransactions),
			client.TransactionCount,
			"Exporter transaction count (%v) ahead of client transaction count (%v)!",
			int(final.TotalTransactions),
			client.TransactionCount,
		)
		assert.LessOrEqualf(
			t,
			int(final.EpochNumber),
			client.Epoch,
			"Exporter epoch (%v) ahead of client epoch (%v)!",
			int(final.EpochNumber),
			client.Epoch,
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
