# Solana Exporter
## Overview

The Solana Exporter exports basic monitoring data from a Solana node, using the 
[Solana RPC API](https://solana.com/docs/rpc).

### Example Usage

To use the Solana Exporter, simply run the program with the desired 
[command line configuration](#Command-Line-Arguments), e.g.,

```shell
solana_exporter \
  -nodekey <VALIDATOR_IDENTITY_1> -nodekey <VALIDATOR_IDENTITY_2> \
  -balance-address <ADDRESS_1> -balance-address <ADDRESS_2> \
  -comprehensive-slot-tracking \
  -monitor-block-sizes
```

## Installation
### Build

Assuming you already have [Go installed](https://go.dev/doc/install), the `solana_exporter` can be installed by 
cloning this repository and building the binary:

```shell
git clone https://github.com/asymmetric-research/solana_exporter.git
cd solana_exporter
CGO_ENABLED=0 go build ./cmd/solana_exporter
```

## Configuration
### Command Line Arguments

The exporter is configured via the following command line arguments:

| Option                         | Description                                                                                                                                                                                                             | Default                   |
|--------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `-balance-address`             | Address to monitor SOL balances for, in addition to the identity and vote accounts of the provided nodekeys - can be set multiple times.                                                                                | N/A                       |
| `-comprehensive-slot-tracking` | Set this flag to track `solana_leader_slots_by_epoch` for all validators.                                                                                                                                               | `false`                   |
| `-http-timeout`                | HTTP timeout to use, in seconds.                                                                                                                                                                                        | `60`                      |
| `-light-mode`                  | Set this flag to enable light-mode. In light mode, only metrics unique to the node being queried are reported (i.e., metrics such as `solana_inflation_rewards` which are visible from any RPC node, are not reported). | `false`                   |
| `-listen-address`              | Prometheus listen address.                                                                                                                                                                                              | `":8080"`                 |
| `-monitor-block-sizes`         | Set this flag to track block sizes (number of transactions) for the configured validators.                                                                                                                              | `false`                   |
| `-nodekey`                     | Solana nodekey (identity account) representing a validator to monitor - can set multiple times.                                                                                                                         | N/A                       |
| `-rpc-url`                     | Solana RPC URL (including protocol and path), e.g., `"http://localhost:8899"` or `"https://api.mainnet-beta.solana.com"`                                                                                                | `"http://localhost:8899"` |
| `-slot-pace`                   | This is the time (in seconds) between slot-watching metric collections                                                                                                                                                  | `1`                       |

### Notes on Configuration

* `-light-mode` is incompatible with both `-monitor-block-sizes` and `-comprehensive-slot-tracking`.
* ***WARNING***:
  * Configuring `-comprehensive-slot-tracking` will lead to potentially thousands of new Prometheus metrics being 
  created every epoch.
  * Configuring `-monitor-block-sizes` with many `-nodekey`'s can potentially strain the node - every block produced 
  by a configured `-nodekey` is fetched, and a typical block can be as large as 5MB.

If you want verbose logs, specify `-v=<num>`. Higher verbosity means more debug output. For most users, the default
verbosity level is fine. If you want detailed log output for missed blocks, run with `-v=1`.

## Metrics
### Overview

The table below describes all the metrics collected by the `solana_exporter`:

| Metric                              | Description                                                                              | Labels                        |
|-------------------------------------|------------------------------------------------------------------------------------------|-------------------------------|
| `solana_validator_active_stake`     | Active stake per validator.                                                              | `votekey`, `nodekey`          |
| `solana_validator_last_vote`        | Last voted-on slot per validator.                                                        | `votekey`, `nodekey`          |
| `solana_validator_root_slot`        | Root slot per validator.                                                                 | `votekey`, `nodekey`          |
| `solana_validator_delinquent`       | Whether a validator is delinquent.                                                       | `votekey`, `nodekey`          |
| `solana_account_balance`            | Solana account balances.                                                                 | `address`                     |
| `solana_node_version`               | Node version of solana.*                                                                 | `version`                     |
| `solana_node_is_healthy`            | Whether the node is healthy.*                                                            | N/A                           |
| `solana_node_num_slots_behind`      | The number of slots that the node is behind the latest cluster confirmed slot.*          | N/A                           |
| `solana_node_minimum_ledger_slot`   | The lowest slot that the node has information about in its ledger.*                      | N/A                           |
| `solana_node_first_available_block` | The slot of the lowest confirmed block that has not been purged from the node's ledger.* | N/A                           |
| `solana_total_transactions`         | Total number of transactions processed without error since genesis.*                     | N/A                           |
| `solana_slot_height`                | The current slot number.*                                                                | N/A                           |
| `solana_epoch_number`               | The current epoch number.*                                                               | N/A                           |
| `solana_epoch_first_slot`           | Current epoch's first slot \[inclusive\].*                                               | N/A                           |
| `solana_epoch_last_slot`            | Current epoch's last slot \[inclusive\].*                                                | N/A                           |
| `solana_leader_slots`               | Number of slots processed.                                                               | `status`, `nodekey`           |
| `solana_leader_slots_by_epoch`      | Number of slots processed.                                                               | `status`, `nodekey`, `epoch`  |
| `solana_inflation_rewards`          | Inflation reward earned.                                                                 | `votekey`, `epoch`            |
| `solana_fee_rewards`                | Transaction fee rewards earned.                                                          | `nodekey`, `epoch`            |
| `solana_block_size`                 | Number of transactions per block.*                                                       | `nodekey`, `transaction_type` |
| `solana_block_height`               | The current block height of the node.                                                    | N/A                           |

***NOTE***: An `*` in the description indicates that the metric **is** tracked in `-light-mode`.

### Labels

The table below describes the various metric labels:

| Label              | Description                         | Options / Example                                    | 
|--------------------|-------------------------------------|------------------------------------------------------|
| `nodekey`          | Validator identity account address. | e.g, `Certusm1sa411sMpV9FPqU5dXAYhmmhygvxJ23S6hJ24`  | 
| `votekey`           | Validator vote account address.     | e.g., `CertusDeBmqN8ZawdkxK5kFGMwBXdudvWHYwtNgNhvLu` |
| `address`          | Solana account address.             | e.g., `Certusm1sa411sMpV9FPqU5dXAYhmmhygvxJ23S6hJ24` |
| `version`          | Solana node version.                | e.g., `v1.18.23`                                     |
| `status`           | Whether a slot was skipped or valid | `valid`, `skipped`                                   |
| `epoch`            | Solana epoch number.                | e.g., `663`                                          |
| `transaction_type` | General transaction type.           | `vote`, `non_vote`                                   |
