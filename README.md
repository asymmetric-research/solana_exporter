# solana_exporter

solana_exporter exports basic monitoring data from a Solana node.

<img src="https://i.imgur.com/2pIXLyU.png" width="550px" alt="" />

Docker images are available on [Docker Hub](https://hub.docker.com/r/certusone/solana_exporter).

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

## Command line arguments

You typically only need to set the RPC URL, pointing to one of your own nodes:

    ./solana_exporter -rpcURI=http://yournode:8899
    
If you want verbose logs, specify `-v=<num>`. Higher verbosity means more debug output. For most users, the default
verbosity level is fine. If you want detailed log output for missed blocks, run with `-v=1`.

```
Usage of solana_exporter:
  -add_dir_header
        If true, adds the file directory to the header of the log messages
  -addr string
        Listen address (default ":8080")
  -alsologtostderr
        log to standard error as well as files
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -log_file string
        If non-empty, use this log file
  -log_file_max_size uint
        Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
  -logtostderr
        log to standard error instead of files (default true)
  -one_output
        If true, only write logs to their native severity level (vs also writing to each lower severity level
  -rpcURI string
        Solana RPC URI (including protocol and path)
  -skip_headers
        If true, avoid header prefixes in the log messages
  -skip_log_headers
        If true, avoid headers when opening log files
  -stderrthreshold value
        logs at or above this threshold go to stderr (default 2)
  -v value
        number for the log level verbosity
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```
