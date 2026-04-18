# Kibana Dev Scripts

Two scripts for running a full Kibana development environment with both
**serverless** and **stateful** instances in parallel.

## Scripts

### `kbn` — Startup Orchestrator

Starts everything from scratch: two Elasticsearch clusters, a shared
optimizer, and two Kibana instances.

```bash
cd /path/to/kibana
kbn                 # start with EIS, Chrome, and tmux log viewer
kbn --clean         # wipe caches and rebuild from scratch
kbn --quiet         # skip the tmux log viewer
```

**What it starts:**

| Component     | Port | Description                     |
| ------------- | ---- | ------------------------------- |
| ES Serverless | 9200 | Docker-based serverless cluster |
| ES Stateful   | 9201 | Snapshot-based trial cluster    |
| Optimizer     | —    | Shared plugin builder (watch)   |
| Kibana SLS    | 5601 | Serverless Kibana               |
| Kibana Stack  | 5611 | Stateful Kibana                 |

**Environment variables:**

| Variable                 | Default                                                | Purpose                       |
| ------------------------ | ------------------------------------------------------ | ----------------------------- |
| `KBN_INFERENCE_URL`      | `https://inference.eu-west-1.aws.svc.qa.elastic.cloud` | EIS URL. Set `""` to disable. |
| `KIBANA_EIS_CCM_API_KEY` | (none)                                                 | Skip vault, use key directly. |
| `CHROME_BIN`             | auto-detected                                          | Path to Chrome binary.        |
| `KBN_LOG_DIR`            | `./logs/kbn-dev`                                       | Log file directory.           |
| `SKIP_BROWSER_LAUNCH`    | (unset)                                                | Set to skip Chrome launch.    |

**Non-interactive mode:** When stdin is not a terminal (piped, CI, agent),
interactive prompts auto-accept and verbose banners are suppressed.

### `kbn-ctl` — Control Plane

Query and control a running `kbn` instance without restarting everything.

```bash
kbn-ctl status            # human-readable health table
kbn-ctl status --json     # machine-readable JSON
kbn-ctl logs kbnsls       # last 50 lines of Kibana Serverless log
kbn-ctl logs all --grep ERROR   # errors across all components
kbn-ctl logs kbnstack --follow  # live tail
kbn-ctl restart kbnsls    # restart Kibana Serverless (ES stays up)
kbn-ctl restart kbnstack  # restart Kibana Stateful
kbn-ctl stop              # stop everything
```

**Components:** `essls`, `esstack`, `optimizer`, `kbnsls`, `kbnstack`, `main`, `all`

## Install

Symlink both scripts into a directory already on your PATH:

```bash
ln -sf "$(pwd)/kbn" /usr/local/bin/kbn
ln -sf "$(pwd)/kbn-ctl" /usr/local/bin/kbn-ctl
```

Or, if you prefer `~/.local/bin` (create it first if it doesn't exist):

```bash
mkdir -p ~/.local/bin
ln -sf "$(pwd)/kbn" ~/.local/bin/kbn
ln -sf "$(pwd)/kbn-ctl" ~/.local/bin/kbn-ctl
```

Verify they're accessible:

```bash
which kbn      # should print /usr/local/bin/kbn or ~/.local/bin/kbn
which kbn-ctl
```

**Prerequisites:** Node.js (nvm), yarn, Docker, Chrome (optional), vault (for EIS).

## Log Viewer

By default, `kbn` automatically opens a tmux session
with four labeled split panes tailing each log file (ES Serverless,
ES Stateful, Kibana SLS, Kibana Stack).

```bash
tmux attach -t kbn-logs   # attach to the log viewer
                           # Detach: Ctrl+B then D
tmux kill-session -t kbn-logs  # close it manually
```

Pass `--quiet` to skip the tmux viewer. Requires `tmux` installed.

## Using with Claude

Both scripts are designed to work with AI coding agents (Claude Code, Cursor).
When run non-interactively:

- `kbn` suppresses the intro banner, auto-accepts Chrome profile creation,
  and skips vault retry loops (continues without EIS).
- `kbn-ctl status --json` returns structured JSON that agents can parse.
- `kbn` writes `status.json` to the log directory at each phase transition.

A Claude Code skill is available at `.claude/skills/kibana-controller/` that
teaches Claude how to start, monitor, and manage the dev environment. The
skill uses dynamic context injection to fetch live status at invocation time.

**Typical agent workflow:**

```bash
# Start (skip tmux viewer, Chrome opens automatically)
kbn --quiet &

# Check status
kbn-ctl status --json

# After code changes — check if Kibana crashed
kbn-ctl status --json   # look for "alive": false

# Restart a crashed instance
kbn-ctl restart kbnsls

# View errors
kbn-ctl logs all --grep "ERROR|FATAL"

# Stop when done
kbn-ctl stop
```

**Available skills:**

- `/kibana-controller` — Start, stop, restart, and monitor Kibana. Claude
  handles polling silently and notifies you when ready.
- `/kbn-status` — Quick status check, presented as a table.

### Auto-cleanup on session end

A Claude Code hook in `.claude/settings.json` runs `kbn-ctl stop` when
your session ends, so ES clusters, Kibana processes, Docker containers,
and Chrome windows are all cleaned up automatically.

To enable this in your own project, add the following to your project's
`.claude/settings.json` (or `~/.claude/settings.json` for all projects):

```json
{
  "hooks": {
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "kbn-ctl stop 2>/dev/null; true"
          }
        ]
      }
    ]
  }
}
```

This requires `kbn-ctl` to be on your PATH (see [Install](#install)).
The `; true` ensures the hook never fails, even if kbn isn't running.
