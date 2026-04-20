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
| `KBN_LOG_DIR`            | `~/.kbn/logs`                                          | Log file directory.            |
| `SKIP_BROWSER_LAUNCH`    | (unset)                                                | Set to skip Chrome launch.    |

**Non-interactive mode:** When stdin is not a terminal (piped, CI, agent),
interactive prompts auto-accept and verbose banners are suppressed.

### `kbn-ctl` — Control Plane

Query and control a `kbn` instance. Can also start it.

```bash
kbn-ctl start                 # start kbn (finds kibana repo automatically)
kbn-ctl start --clean --quiet # clean start, no tmux viewer
kbn-ctl status                # human-readable health table
kbn-ctl status --json         # machine-readable JSON
kbn-ctl logs kbnsls           # last 50 lines of Kibana Serverless log
kbn-ctl logs all --grep ERROR # errors across all components
kbn-ctl logs kbnstack --follow  # live tail
kbn-ctl attach                # attach to the tmux log viewer
kbn-ctl restart kbnsls        # restart Kibana Serverless (ES stays up)
kbn-ctl restart kbnstack      # restart Kibana Stateful
kbn-ctl stop                  # stop everything
kbn-ctl run yarn kbn bootstrap  # run any command with correct node/nvm
kbn-ctl run node --version      # check which node version is active
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

## Manual setup (without these scripts)

If you want to run both serverless and stateful Kibana side-by-side
without `kbn` or `kbn-ctl`, here's what you need to do manually.
This is what the scripts automate.

### 1. Switch to the correct Node version

```bash
cd /path/to/kibana
nvm use    # reads .nvmrc
```

### 2. Bootstrap (if needed)

```bash
yarn kbn bootstrap
```

### 3. Clean up stale Docker containers

Previous failed serverless runs leave containers behind that block
the next start:

```bash
docker rm -f es01 es02 es03 uiam uiam-cosmosdb 2>/dev/null
```

### 4. Start ES Serverless (Terminal 1)

```bash
yarn es serverless \
  --projectType elasticsearch_general_purpose \
  --clean --kill
```

Wait for: `succ Serverless ES cluster running`

### 5. Start ES Stateful (Terminal 2)

Must use different ports to avoid conflicts with serverless:

```bash
yarn es snapshot \
  --license trial --clean \
  -E http.port=9201 \
  -E transport.port=9301 \
  -E xpack.ml.enabled=false
```

Wait for: `succ ES cluster is ready`

### 6. Start the shared optimizer (Terminal 3)

Running two Kibana instances each with their own optimizer uses
~8-16GB RAM. A single shared optimizer avoids this:

```bash
node scripts/build_kibana_platform_plugins --watch
```

Wait for: `succ all bundles cached` or `succ ... bundles compiled successfully`

### 7. (Optional) Run EIS setup

If using Elastic Inference Service, run after each ES cluster is ready:

```bash
# Requires vault login first:
# VAULT_ADDR=https://secrets.elastic.co:8200 vault login -method oidc
node scripts/eis.js
```

### 8. Start Kibana Serverless (Terminal 4)

```bash
yarn serverless-es \
  --server.port=5601 \
  --no-optimizer
```

### 9. Start Kibana Stateful (Terminal 5)

Must use a different port and cookie name to avoid auth conflicts
with the serverless instance:

```bash
yarn start \
  --config config/kibana.stack.dev.yml \
  --server.port=5611 \
  --xpack.security.cookieName=sid-stack \
  --no-optimizer
```

### 10. Access

- Serverless: http://localhost:5601 (select admin role)
- Stateful: http://localhost:5611 (login: elastic / changeme)

### Gotchas to know

- **Startup order matters.** ES must be ready before Kibana starts.
- **Cookie conflict.** Without `--xpack.security.cookieName=sid-stack`
  on stateful, logging into one instance logs you out of the other.
- **Optimizer memory.** Without `--no-optimizer` on both + a shared
  optimizer, each Kibana spawns its own (~4-8GB each).
- **Port conflicts.** If you don't set `-E http.port=9201` and
  `-E transport.port=9301` on stateful ES, it clashes with serverless.
- **Docker cleanup.** Stale `uiam`/`es0x` containers from a previous
  serverless run cause the next one to fail with "unhealthy" errors.
- **Orphaned processes.** If Kibana crashes, `node` processes stay on
  ports 5601/5611 and block the next start. Kill them with
  `lsof -ti tcp:5601 | xargs kill`.
- **Branch switches.** After switching branches, you usually need
  `yarn kbn clean && yarn kbn bootstrap` and a fresh start.

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

