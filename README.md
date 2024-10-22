# solana_exporter

solana_exporter exports basic monitoring data from a Solana node.

<img src="https://i.imgur.com/2pIXLyU.png" width="550px" alt="" />

## Metrics

Metrics tracked with confirmation level `recent`:

- **solana_validator_root_slot** - Latest root seen by each validator.
- **solana_validator_last_vote** - Latest vote by each validator (not necessarily on the majority fork!)
- **solana_validator_delinquent** - Whether node considers each validator to be delinquent.
- **solana_validator_activated_stake**  - Active stake for each validator.
- **solana_active_validators** - Total number of active/delinquent validators.

Metrics tracked with confirmation level `max`:

- **solana_leader_slots_total** - Number of leader slots per leader, grouped by skip status.
- **solana_confirmed_epoch_first_slot** - Current epoch's first slot.
- **solana_confirmed_epoch_last_slot** - Current epoch's last slot.
- **solana_confirmed_epoch_number** - Current epoch.
- **solana_confirmed_slot_height** - Last confirmed slot height observed.
- **solana_confirmed_transactions_total** - Total number of transactions processed since genesis.

Metrics with no confirmation level:

- **solana_node_version** - Current solana-validator node version.

## Installation

`solana_exporter` can be installed by doing the following. It's assumed you already have `go` installed.

```sh
git clone https://github.com/asymmetric-research/solana_exporter.git
cd solana_exporter
CGO_ENABLED=0 go build ./cmd/solana_exporter
```

## Command line arguments

You typically only need to set the RPC URL, pointing to one of your own nodes:

    ./solana_exporter -rpc-url=http://yournode:8899

If you want verbose logs, specify `-v=<num>`. Higher verbosity means more debug output. For most users, the default
verbosity level is fine. If you want detailed log output for missed blocks, run with `-v=1`.

```
Usage of solana_exporter:
  -add_dir_header
    	If true, adds the file directory to the header of the log messages
  -alsologtostderr
    	log to standard error as well as files (no effect when -logtostderr=true)
  -balance-address value
    	Address to monitor SOL balances for, in addition to the identity and vote accounts of the provided nodekeys - can be set multiple times.
  -comprehensive-slot-tracking
    	Set this flag to track solana_leader_slots_by_epoch for ALL validators. Warning: this will lead to potentially thousands of new Prometheus metrics being created every epoch.
  -http-timeout int
    	HTTP timeout to use, in seconds. (default 60)
  -listen-address string
    	Listen address (default ":8080")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory (no effect when -logtostderr=true)
  -log_file string
    	If non-empty, use this log file (no effect when -logtostderr=true)
  -log_file_max_size uint
    	Defines the maximum size a log file can grow to (no effect when -logtostderr=true). Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
  -logtostderr
    	log to standard error instead of files (default true)
  -monitor-block-sizes
    	Set this flag to track block sizes (number of transactions) for the configured validators. Warning: this might grind the RPC node.
  -nodekey value
    	Solana nodekey (identity account) representing validator to monitor - can set multiple times.
  -one_output
    	If true, only write logs to their native severity level (vs also writing to each lower severity level; no effect when -logtostderr=true)
  -rpc-url string
    	Solana RPC URL (including protocol and path), e.g., 'http://localhost:8899' or 'https://api.mainnet-beta.solana.com' (default "http://localhost:8899")
  -skip_headers
    	If true, avoid header prefixes in the log messages
  -skip_log_headers
    	If true, avoid headers when opening log files (no effect when -logtostderr=true)
  -stderrthreshold value
    	logs at or above this threshold go to stderr when writing to files and stderr (no effect when -logtostderr=true or -alsologtostderr=true) (default 2)
  -v value
    	number for the log level verbosity
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```
