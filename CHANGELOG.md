# v3.0.0

## Key Changes

The new `solana-exporter` (renamed from `solana_exporter`) contains many new metrics, standardised naming conventions 
and more configurability.

## What's Changed
### Metric Updates
#### New Metrics

Below is a list of newly added metrics (see the [README](README.md) 
for metric descriptions):

* `solana_account_balance` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)
* `solana_node_is_healthy` (<u><strong>[@GranderStark](https://github.com/GranderStark)</strong></u>)
* `solana_nude_num_slots_behind` (<u><strong>[@GranderStark](https://github.com/GranderStark)</strong></u>)
* `solana_node_minimum_ledger_slot` (<u><strong>[@GranderStark](https://github.com/GranderStark)</strong></u>)
* `solana_node_first_available_block` (<u><strong>[@GranderStark](https://github.com/GranderStark)</strong></u>)
* `solana_cluster_slots_by_epoch_total` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>) 
* `solana_validator_fee_rewards` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)
* `solana_validator_block_size` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)
* `solana_node_block_height` (<u><strong>[@GranderStark](https://github.com/GranderStark)</strong></u>)
* `solana_cluster_active_stake` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)
* `solana_cluster_last_vote` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)
* `solana_cluster_root_slot` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)
* `solana_cluster_validator_count` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)

#### Renamed Metrics

The table below contains all metrics renamed in `v3.0.0` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>):

| Old Name                              | New Name                                       |
|---------------------------------------|------------------------------------------------|
| `solana_validator_activated_stake`    | `solana_validator_active_stake`                |
| `solana_confirmed_transactions_total` | `solana_node_transactions_total`               |
| `solana_confirmed_slot_height`        | `solana_node_slot_height`                      |
| `solana_confirmed_epoch_number`       | `solana_node_epoch_number`                     |
| `solana_confirmed_epoch_first_slot`   | `solana_node_epoch_first_slot`                 |
| `solana_confirmed_epoch_last_slot`    | `solana_node_epoch_last_slot`                  |
| `solana_leader_slots_total`           | `solana_validator_leader_slots_total`          |
| `solana_leader_slots_by_epoch`        | `solana_validator_leader_slots_by_epoch_total` |
| `solana_active_validators`            | `solana_cluster_validator_count`               |

Metrics were renamed to:
* Remove commitment levels from metric names.
* Standardise naming conventions:
  * `solana_validator_*`: Validator-specific metrics which are trackable from any RPC node (i.e., active stake).
  * `solana_node_*`: Node-specific metrics which are not trackable from other nodes (i.e., node health).

#### Label Updates

The following labels were renamed (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>):
 * `pubkey` was renamed to `votekey`, to clearly identity that it refers to the address of a validators vote account.

### Config Updates
#### New Config Parameters

Below is a list of newly added config parameters (see the [README](README.md) 
for parameter descriptions) (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>):

 * `-balance-address`
 * `-nodekey`
 * `-comprehensive-slot-tracking`
 * `-monitor-block-sizes`
 * `-slot-pace`
 * `-light-mode`
 * `-http-timeout`
 * `-comprehensive-vote-account-tracking`

#### Renamed Config Parameters

The table below contains all config parameters renamed in `v3.0.0` (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>):

| Old Name                            | New Name          |
|-------------------------------------|-------------------|
| `-rpcURI`                           | `-rpc-url`        |
| `addr`                              | `-listen-address` |

#### Removed Config Parameters

The following metrics were removed (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>):

 * `votepubkey`. Configure validator tracking using the `-nodekey` parameter.

### General Updates

* The project was renamed from `solana_exporter` to `solana-exporter`, to conform with 
[Go naming conventions](https://github.com/unknwon/go-code-convention/blob/main/en-US.md) (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>).
* Testing was significantly improved (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>).
* [klog](https://github.com/kubernetes/klog) logging was removed and replaced with [zap](https://github.com/uber-go/zap)
  (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>)
* Easy usage (<u><strong>[@johnstonematt](https://github.com/johnstonematt)</strong></u>):
  * The example dashboard was updated.
  * An example prometheus config was added, as well as recording rules for tracking skip rate.

## New Contributors

* <u><strong>[@GranderStark](https://github.com/GranderStark)</strong></u> made their first contribution.
