# Commands

Stash for random scripts and utilities I try out.

## Kibana Process Orchestrator

A Go-based CLI tool for managing Kibana and Elasticsearch development processes with an interactive menu interface.

### Features

- **Interactive Menu**: Easy-to-use terminal UI for managing processes
- **Dependency-Aware**: Understands process dependencies and handles restarts correctly
- **Process Monitoring**: Real-time metrics (CPU, memory, threads) for running processes
- **Log Management**: View, tail, and open log files directly from the menu
- **Configurable**: Customizable via `.env` file or environment variables

### Process Dependency Graph

```
optimizer (no dependencies)
  └─> kbnsls (depends on: optimizer, essls)
  └─> kbnstack (depends on: optimizer, esstack)

essls (no dependencies)
  └─> kbnsls (depends on: optimizer, essls)

esstack (no dependencies)
  └─> kbnstack (depends on: optimizer, esstack)

kbnsls (depends on: optimizer, essls)
  └─> sls_chrome (depends on: kbnsls)

kbnstack (depends on: optimizer, esstack)
  └─> stack_chrome (depends on: kbnstack)
```

### Installation

```bash
# Build the orchestrator
cd /path/to/commands
go build -o bin/kbn-orchestrator ./cmd/kbn/

# Run it
./bin/kbn
```

### Configuration

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key configuration options:

| Variable | Description | Default |
|----------|-------------|---------|
| `KIBANA_DIR` | Path to Kibana repository | `~/workplace/kibana` |
| `LOG_DIR` | Directory for log files | `~/workplace/local_dev_logs` |
| `BROWSER_PATH` | Path to browser executable | Chrome on macOS |
| `LOG_OPEN_CMD` | Command to open logs | `open` |

### Usage

Run `./bin/kbn` to start the interactive menu:

```
============================================================
  Kibana Process Orchestrator
============================================================

  Process         Status     PID        Info
  -------------------------------------------------------
  optimizer       ready      12345      CPU: 2.1% MEM: 512MB
  essls           ready      12346      CPU: 5.3% MEM: 2048MB
  ...

------------------------------------------------------------
Select action:
> Start all processes
  Stop all processes
  Restart process
  Stop process
  View process details
  View logs
  Refresh status
  Exit
```

### Menu Options

1. **Start all processes** - Starts all processes in dependency order
2. **Stop all processes** - Stops all running processes
3. **Restart process** - Select and restart a process (and its dependents)
4. **Stop process** - Stop a single process
5. **View process details** - See metrics, child processes, and process info
6. **View logs** - Open or tail log files
7. **Exit** - Option to kill all processes or leave them running

### Project Structure

```
commands/
├── bin/
│   ├── kbn                 # Launcher script
│   └── kbn-orchestrator    # Go binary
├── cmd/
│   └── kbn/
│       └── main.go         # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go       # Configuration loading
│   └── orchestrator/
│       ├── orchestrator.go # Core orchestration logic
│       ├── process.go      # Process management
│       ├── dependencies.go # Dependency resolution
│       ├── metrics.go      # Process metrics collection
│       ├── logs.go         # Log file management
│       └── state.go        # Process state tracking
├── scripts/
│   ├── process_optimizer.sh
│   ├── process_essls.sh
│   ├── process_esstack.sh
│   ├── process_kbnsls.sh
│   ├── process_kbnstack.sh
│   └── process_chrome.sh
├── .env.example            # Example configuration
├── go.mod
└── go.sum
```

### Original Bash Script

The original bash implementation is preserved as `bin/kbn.bak` for reference.
