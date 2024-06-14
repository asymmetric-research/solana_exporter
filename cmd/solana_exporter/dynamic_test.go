package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSolanaCollector_Collect_Dynamic(t *testing.T) {
	client := newDynamicRPCClient()
	collector := createSolanaCollector(
		client,
		slotPacerSchedule,
	)
	prometheus.NewPedanticRegistry().MustRegister(collector)

	// start off by testing initial state:
	testCases := []collectionTest{
		{
			Name: "solana_active_validators",
			ExpectedResponse: `
# HELP solana_active_validators Total number of active validators by state
# TYPE solana_active_validators gauge
solana_active_validators{state="current"} 3
solana_active_validators{state="delinquent"} 0
`,
		},
		{
			Name: "solana_validator_activated_stake",
			ExpectedResponse: `
# HELP solana_validator_activated_stake Activated stake per validator
# TYPE solana_validator_activated_stake gauge
solana_validator_activated_stake{nodekey="aaa",pubkey="AAA"} 1000000
solana_validator_activated_stake{nodekey="bbb",pubkey="BBB"} 1000000
solana_validator_activated_stake{nodekey="ccc",pubkey="CCC"} 1000000
`,
		},
		{
			Name: "solana_validator_root_slot",
			ExpectedResponse: `
# HELP solana_validator_root_slot Root slot per validator
# TYPE solana_validator_root_slot gauge
solana_validator_root_slot{nodekey="aaa",pubkey="AAA"} 0
solana_validator_root_slot{nodekey="bbb",pubkey="BBB"} 0
solana_validator_root_slot{nodekey="ccc",pubkey="CCC"} 0
`,
		},
		{
			Name: "solana_validator_delinquent",
			ExpectedResponse: `
# HELP solana_validator_delinquent Whether a validator is delinquent
# TYPE solana_validator_delinquent gauge
solana_validator_delinquent{nodekey="aaa",pubkey="AAA"} 0
solana_validator_delinquent{nodekey="bbb",pubkey="BBB"} 0
solana_validator_delinquent{nodekey="ccc",pubkey="CCC"} 0
`,
		},
		{
			Name: "solana_node_version",
			ExpectedResponse: `
# HELP solana_node_version Node version of solana
# TYPE solana_node_version gauge
solana_node_version{version="v1.0.0"} 1
`,
		},
	}

	runCollectionTests(t, collector, testCases)

	// now make some changes:
	client.UpdateStake("aaa", 2_000_000)
	client.UpdateStake("bbb", 500_000)
	client.UpdateDelinquency("ccc", true)
	client.UpdateVersion("v1.2.3")

	// now test the final state
	testCases = []collectionTest{
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
solana_validator_activated_stake{nodekey="aaa",pubkey="AAA"} 2000000
solana_validator_activated_stake{nodekey="bbb",pubkey="BBB"} 500000
solana_validator_activated_stake{nodekey="ccc",pubkey="CCC"} 1000000
`,
		},
		{
			Name: "solana_validator_root_slot",
			ExpectedResponse: `
# HELP solana_validator_root_slot Root slot per validator
# TYPE solana_validator_root_slot gauge
solana_validator_root_slot{nodekey="aaa",pubkey="AAA"} 0
solana_validator_root_slot{nodekey="bbb",pubkey="BBB"} 0
solana_validator_root_slot{nodekey="ccc",pubkey="CCC"} 0
`,
		},
		{
			Name: "solana_validator_delinquent",
			ExpectedResponse: `
# HELP solana_validator_delinquent Whether a validator is delinquent
# TYPE solana_validator_delinquent gauge
solana_validator_delinquent{nodekey="aaa",pubkey="AAA"} 0
solana_validator_delinquent{nodekey="bbb",pubkey="BBB"} 0
solana_validator_delinquent{nodekey="ccc",pubkey="CCC"} 1
`,
		},
		{
			Name: "solana_node_version",
			ExpectedResponse: `
# HELP solana_node_version Node version of solana
# TYPE solana_node_version gauge
solana_node_version{version="v1.2.3"} 1
`,
		},
	}

	runCollectionTests(t, collector, testCases)
}

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

func TestSolanaCollector_WatchSlots_Dynamic(t *testing.T) {
	// reset metrics before running tests:
	leaderSlotsTotal.Reset()
	leaderSlotsByEpoch.Reset()

	// create clients:
	client := newDynamicRPCClient()
	collector := createSolanaCollector(
		client,
		300*time.Millisecond,
	)
	prometheus.NewPedanticRegistry().MustRegister(collector)

	// start client/collector and wait a bit:
	runCtx, runCancel := context.WithCancel(context.Background())
	go client.Run(runCtx)
	time.Sleep(time.Second)
	slotsCtx, slotsCancel := context.WithCancel(context.Background())
	go collector.WatchSlots(slotsCtx)
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

	// cancel and wait for cancellation:
	slotsCancel()
	runCancel()
	time.Sleep(time.Second)
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
