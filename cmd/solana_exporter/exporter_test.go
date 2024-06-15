package main

import (
	"bytes"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var staticCollector = createSolanaCollector(&staticRPCClient{})

func TestSolanaCollector_Collect(t *testing.T) {
	prometheus.NewPedanticRegistry().MustRegister(staticCollector)

	testCases := map[string]string{
		"solana_active_validators": `
# HELP solana_active_validators Total number of active validators by state
# TYPE solana_active_validators gauge
solana_active_validators{state="current"} 2
solana_active_validators{state="delinquent"} 1
`,
		"solana_validator_activated_stake": `
# HELP solana_validator_activated_stake Activated stake per validator
# TYPE solana_validator_activated_stake gauge
solana_validator_activated_stake{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 49
solana_validator_activated_stake{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 42
solana_validator_activated_stake{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 43
`,
		"solana_validator_last_vote": `
# HELP solana_validator_last_vote Last voted slot per validator
# TYPE solana_validator_last_vote gauge
solana_validator_last_vote{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 92
solana_validator_last_vote{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 147
solana_validator_last_vote{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 148
`,
		"solana_validator_root_slot": `
# HELP solana_validator_root_slot Root slot per validator
# TYPE solana_validator_root_slot gauge
solana_validator_root_slot{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 3
solana_validator_root_slot{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 18
solana_validator_root_slot{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 19
`,
		"solana_validator_delinquent": `
# HELP solana_validator_delinquent Whether a validator is delinquent
# TYPE solana_validator_delinquent gauge
solana_validator_delinquent{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 1
solana_validator_delinquent{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 0
solana_validator_delinquent{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 0
`,
		"solana_node_version": `
# HELP solana_node_version Node version of solana
# TYPE solana_node_version gauge
solana_node_version{version="1.16.7"} 1
`,
	}

	for testName, expectedValue := range testCases {
		t.Run(
			testName,
			func(t *testing.T) {
				if err := testutil.CollectAndCompare(
					staticCollector,
					bytes.NewBufferString(expectedValue),
					testName,
				); err != nil {
					t.Errorf("unexpected collecting result for %s: \n%s", testName, err)
				}
			},
		)
	}
}

func TestSolanaCollector_WatchSlots(t *testing.T) {
	go staticCollector.WatchSlots()
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

	hosts := []string{
		"B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",
		"C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",
		"4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",
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
					for _, host := range hosts {
						testBlockProductionMetric(t, metric, host, status)
					}
				})
			}
		})
	}
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
	assert.Equal(t, expectedValue, testutil.ToFloat64(metric.WithLabelValues(labels...)))
}
