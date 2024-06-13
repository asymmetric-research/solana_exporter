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

func TestSolanaCollector_Collect_Static(t *testing.T) {
	collector := createSolanaCollector(
		&staticRPCClient{},
		slotPacerSchedule,
	)
	prometheus.NewPedanticRegistry().MustRegister(collector)

	testCases := []collectionTest{
		{
			Name: "solana_active_validators",
			ExpectedResponse: `
# HELP solana_active_validators Total number of active validators by state
# TYPE solana_active_validators gauge
solana_active_validators{state="current"} 2
solana_active_validators{state="delinquent"} 1
`,
		},
		{
			Name: "solana_validator_activated_stake",
			ExpectedResponse: `
# HELP solana_validator_activated_stake Activated stake per validator
# TYPE solana_validator_activated_stake gauge
solana_validator_activated_stake{nodekey="aaa",pubkey="AAA"} 49
solana_validator_activated_stake{nodekey="bbb",pubkey="BBB"} 42
solana_validator_activated_stake{nodekey="ccc",pubkey="CCC"} 43
`,
		},
		{
			Name: "solana_validator_last_vote",
			ExpectedResponse: `
# HELP solana_validator_last_vote Last voted slot per validator
# TYPE solana_validator_last_vote gauge
solana_validator_last_vote{nodekey="aaa",pubkey="AAA"} 92
solana_validator_last_vote{nodekey="bbb",pubkey="BBB"} 147
solana_validator_last_vote{nodekey="ccc",pubkey="CCC"} 148
`,
		},
		{
			Name: "solana_validator_root_slot",
			ExpectedResponse: `
# HELP solana_validator_root_slot Root slot per validator
# TYPE solana_validator_root_slot gauge
solana_validator_root_slot{nodekey="aaa",pubkey="AAA"} 3
solana_validator_root_slot{nodekey="bbb",pubkey="BBB"} 18
solana_validator_root_slot{nodekey="ccc",pubkey="CCC"} 19
`,
		},
		{
			Name: "solana_validator_delinquent",
			ExpectedResponse: `
# HELP solana_validator_delinquent Whether a validator is delinquent
# TYPE solana_validator_delinquent gauge
solana_validator_delinquent{nodekey="aaa",pubkey="AAA"} 1
solana_validator_delinquent{nodekey="bbb",pubkey="BBB"} 0
solana_validator_delinquent{nodekey="ccc",pubkey="CCC"} 0
`,
		},
		{
			Name: "solana_node_version",
			ExpectedResponse: `
# HELP solana_node_version Node version of solana
# TYPE solana_node_version gauge
solana_node_version{version="1.16.7"} 1
`,
		},
	}

	runCollectionTests(t, collector, testCases)
}

func TestSolanaCollector_WatchSlots_Static(t *testing.T) {
	// reset metrics before running tests:
	leaderSlotsTotal.Reset()
	leaderSlotsByEpoch.Reset()

	collector := createSolanaCollector(
		&staticRPCClient{},
		100*time.Millisecond,
	)
	prometheus.NewPedanticRegistry().MustRegister(collector)
	ctx, cancel := context.WithCancel(context.Background())
	go collector.WatchSlots(ctx)
	time.Sleep(1 * time.Second)

	tests := []struct {
		expectedValue float64
		metric        prometheus.Gauge
	}{
		{
			expectedValue: float64(staticEpochInfo.AbsoluteSlot),
			metric:        confirmedSlotHeight,
		},
		{
			expectedValue: float64(staticEpochInfo.TransactionCount),
			metric:        totalTransactionsTotal,
		},
		{
			expectedValue: float64(staticEpochInfo.Epoch),
			metric:        currentEpochNumber,
		},
		{
			expectedValue: float64(staticEpochInfo.AbsoluteSlot - staticEpochInfo.SlotIndex),
			metric:        epochFirstSlot,
		},
		{
			expectedValue: float64(staticEpochInfo.AbsoluteSlot - staticEpochInfo.SlotIndex + staticEpochInfo.SlotsInEpoch),
			metric:        epochLastSlot,
		},
	}

	for _, testCase := range tests {
		name := extractName(testCase.metric.Desc())
		t.Run(
			name,
			func(t *testing.T) {
				assert.Equal(t, testCase.expectedValue, testutil.ToFloat64(testCase.metric))
			},
		)
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
	// cancel and wait for cancellation:
	cancel()
	time.Sleep(time.Second)
}

func testBlockProductionMetric(
	t *testing.T,
	metric *prometheus.CounterVec,
	host string,
	status string,
) {
	hostInfo := staticBlockProduction.Hosts[host]
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
