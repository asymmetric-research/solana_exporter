package main

import (
	"bytes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"testing"
)

var staticCollector = createSolanaCollector(&staticRPCClient{})

func TestSolanaCollector_Collect(t *testing.T) {
	prometheus.NewPedanticRegistry().MustRegister(staticCollector)

	t.Run(
		"TestTotalValidators",
		func(t *testing.T) {
			expectedTotalValidators := `
# HELP solana_active_validators Total number of active validators by state
# TYPE solana_active_validators gauge
solana_active_validators{state="current"} 2
solana_active_validators{state="delinquent"} 1
`
			if err := testutil.CollectAndCompare(
				staticCollector,
				bytes.NewBufferString(expectedTotalValidators),
				"solana_active_validators",
			); err != nil {
				t.Errorf("unexpected collecting result for total validators:\n%s", err)
			}
		},
	)

	t.Run(
		"TestActivatedStake",
		func(t *testing.T) {
			expectedActivatedStake := `
# HELP solana_validator_activated_stake Activated stake per validator
# TYPE solana_validator_activated_stake gauge
solana_validator_activated_stake{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 49
solana_validator_activated_stake{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 42
solana_validator_activated_stake{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 43
`
			if err := testutil.CollectAndCompare(
				staticCollector,
				bytes.NewBufferString(expectedActivatedStake),
				"solana_validator_activated_stake",
			); err != nil {
				t.Errorf("unexpected collecting result for activated stake:\n%s", err)
			}
		},
	)

	t.Run(
		"TestLastVote",
		func(t *testing.T) {
			expectedLastVote := `
# HELP solana_validator_last_vote Last voted slot per validator
# TYPE solana_validator_last_vote gauge
solana_validator_last_vote{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 92
solana_validator_last_vote{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 147
solana_validator_last_vote{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 148
`
			if err := testutil.CollectAndCompare(
				staticCollector,
				bytes.NewBufferString(expectedLastVote),
				"solana_validator_last_vote",
			); err != nil {
				t.Errorf("unexpected collecting result for last vote:\n%s", err)
			}
		},
	)

	t.Run(
		"TestRootSlot",
		func(t *testing.T) {
			expectedRootSlot := `
# HELP solana_validator_root_slot Root slot per validator
# TYPE solana_validator_root_slot gauge
solana_validator_root_slot{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 3
solana_validator_root_slot{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 18
solana_validator_root_slot{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 19
`
			if err := testutil.CollectAndCompare(
				staticCollector,
				bytes.NewBufferString(expectedRootSlot),
				"solana_validator_root_slot",
			); err != nil {
				t.Errorf("unexpected collecting result for root slot:\n%s", err)
			}
		},
	)

	t.Run(
		"TestValidatorDelinquent",
		func(t *testing.T) {
			expectedRootSlot := `
# HELP solana_validator_delinquent Whether a validator is delinquent
# TYPE solana_validator_delinquent gauge
solana_validator_delinquent{nodekey="4MUdt8D2CadJKeJ8Fv2sz4jXU9xv4t2aBPpTf6TN8bae",pubkey="xKUz6fZ79SXnjGYaYhhYTYQBoRUBoCyuDMkBa1tL3zU"} 1
solana_validator_delinquent{nodekey="B97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="3ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 0
solana_validator_delinquent{nodekey="C97CCUW3AEZFGy6uUg6zUdnNYvnVq5VG8PUtb2HayTDD",pubkey="4ZT31jkAGhUaw8jsy4bTknwBMP8i4Eueh52By4zXcsVw"} 0
`
			if err := testutil.CollectAndCompare(
				staticCollector,
				bytes.NewBufferString(expectedRootSlot),
				"solana_validator_delinquent",
			); err != nil {
				t.Errorf("unexpected collecting result for validator delinquent:\n%s", err)
			}
		},
	)

	t.Run(
		"TestNodeVersion",
		func(t *testing.T) {
			expectedRootSlot := `
# HELP solana_node_version Node version of solana
# TYPE solana_node_version gauge
solana_node_version{version="1.16.7"} 1

`
			if err := testutil.CollectAndCompare(
				staticCollector,
				bytes.NewBufferString(expectedRootSlot),
				"solana_node_version",
			); err != nil {
				t.Errorf("unexpected collecting result for node version:\n%s", err)
			}
		},
	)
}
