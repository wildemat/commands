# kbn + kbn-ctl Demo Script

## Overview (30 seconds)

Two bash scripts that replace the manual multi-terminal Kibana dev setup:
- **`kbn`** — one command to start everything (2x ES, optimizer, 2x Kibana, Chrome)
- **`kbn-ctl`** — control plane to check status, view logs, restart, stop

Plus a **Claude Code skill** that gives Claude full control over the dev environment.

---

## Quick demo (5 minutes)

Pre-req: have Kibana already running from a `kbn` invocation before the demo
starts (saves the 3-5 min startup wait).

### 1. Show it's running (30s)

```bash
kbn-ctl status
tmux attach -t kbn-logs   # flash the 4-pane viewer, then Ctrl+B D to detach
```

"One command started all of this. Two ES clusters, optimizer, two Kibanas,
Chrome windows, and a tmux log viewer — all managed as one unit."

### 2. Control it (30s)

```bash
kbn-ctl restart kbnstack              # restart stateful Kibana
kbn-ctl logs kbnstack --tail 5        # show it restarting
kbn-ctl status                        # show it come back up
```

"kbn-ctl gives you a control plane without hunting for PIDs or terminals."

### 3. Claude starts Kibana (1.5 min)

Stop it first, then have Claude start it:

```bash
kbn-ctl stop
```

In Claude Code:
```
> start kibana
```

Show: Claude loads the skill, starts `kbn --quiet` in background, polls
silently, reports when ready. Chrome opens.

### 4. Claude checks the UI (1 min)

```
> is the getting started page working on serverless?
```

Show: Claude uses browser tools or curl with auth to check the page.
On serverless it picks the admin role. On stateful it uses elastic/changeme.

### 5. New session awareness (30s)

Start a fresh Claude session:
```
> is kibana running?
```

Show: Claude detects the existing instance, reports URLs — no restart.

### 6. The pitch (30s)

"Before: 5 terminals, manual yarn commands, killing orphan PIDs, Docker
cleanup. Now: one command for humans, one skill for Claude. No hardcoded
paths, works on any machine, shared via git."

---

## Full demo (15-20 minutes)

## Part 1: Human workflow (manual usage)

### 1.1 Start everything from scratch

```bash
cd ~/workplace/kibana
kbn
```

**Show:**
- Intro banner with config summary (repo, EIS, clean, browser, show-logs)
- LOGGING section — tmux viewer opens automatically with labeled panes
- Stale Docker container cleanup runs first
- Dev ports cleared before ES starts
- ES Serverless + Stateful start in parallel
- Bootstrap + optimizer run while ES boots
- Kibana instances start as ES clusters become ready
- Two Chrome windows open with separate profiles (kbn-sls, kbn-stack)
- Final STATUS block with OK/FAILED, PIDs, and log viewer attach command

### 1.2 tmux log viewer

```bash
tmux attach -t kbn-logs
```

**Show:**
- 4 labeled split panes: ES Serverless, ES Stateful, Kibana SLS, Kibana Stack
- Pane titles visible at the top of each pane
- Detach with Ctrl+B then D

### 1.3 kbn-ctl status

```bash
kbn-ctl status           # human table
kbn-ctl status --json    # machine-readable
```

**Show:**
- Table with COMPONENT / PROCESS / READY / PORT columns
- JSON output with alive, ready, port_open for each component

### 1.4 kbn-ctl logs

```bash
kbn-ctl logs kbnsls --tail 10
kbn-ctl logs all --grep ERROR
kbn-ctl logs essls --follow    # live tail
```

### 1.5 Restart a Kibana instance

```bash
kbn-ctl restart kbnstack
```

**Show:**
- Kills the Kibana process on port 5611
- Monitor loop in kbn auto-restarts it
- `kbn-ctl status` shows it come back up

### 1.6 Stop everything

```bash
kbn-ctl stop
```

**Show:**
- Chrome windows close
- Kibana processes killed by port
- Docker containers (es01-03, uiam, uiam-cosmosdb) stopped and removed
- tmux session killed
- Clean shutdown message

### 1.7 Clean start (after branch switch)

```bash
kbn --clean
```

**Show:**
- Wipes `.es/cache`
- Removes all kibana-ci Docker images (re-pulls fresh)
- Runs `yarn kbn clean` + `yarn kbn bootstrap`
- Full rebuild from scratch

### 1.8 Quiet mode (for scripts/CI)

```bash
kbn --quiet
```

- Skips tmux viewer, everything else the same

---

## Part 2: Non-interactive / agent mode

### 2.1 How it detects non-interactive

- `[ -t 0 ]` — checks if stdin is a terminal
- When piped or run by an agent: suppresses intro banner, auto-accepts
  Chrome profile creation, skips vault retry loop

### 2.2 status.json for tooling

```bash
cat ~/workplace/kibana/logs/kbn-dev/status.json
```

**Show:**
- Written at each phase: starting → es_starting → optimizer_ready → running
- Components with PIDs, log paths, ports
- `kbn-ctl` reads this + live process/port checks

---

## Part 3: Claude Code skill demo

### 3.1 Start Kibana via Claude

```
> start up kibana
```

**Show Claude:**
- Loads `/kibana-controller` skill automatically
- Checks current status first (detects if already running)
- Says "Spinning up serverless and stateful Kibana, standby..."
- Runs `kbn --quiet` in background
- Polls silently — no noisy intermediate output
- Reports when ready with URLs
- Chrome windows open automatically

### 3.2 Check status mid-session

```
> /kbn-status
```

**Show:**
- Clean table output, no raw JSON
- Shows which components are up/ready

### 3.3 Ask about running UI

```
> is the getting started page working on serverless?
```

**Show Claude:**
- Uses browser MCP or curl to check the page
- Logs into serverless with admin role
- Reports what it sees

### 3.4 Stateful browser interaction

```
> check the dev tools console on stateful
```

**Show Claude:**
- Navigates to http://localhost:5611/app/dev_tools#/console
- Logs in with elastic / changeme
- Reports the page state

### 3.5 Partial failure recovery

```
> start kibana
```
(when uiam is failing)

**Show Claude:**
- Stateful comes up, serverless fails
- Claude automatically retries serverless: cleans containers, restarts ES, launches Kibana SLS
- Reports partial success, then full success after recovery

### 3.6 Restart after code changes

**Make a code change, then:**
```
> kibana seems broken, can you restart it?
```

**Show Claude:**
- Runs `kbn-ctl restart kbnsls` / `kbn-ctl restart kbnstack`
- Polls until ready
- Reports back

### 3.7 New session picks up existing instance

**Start a new Claude session:**
```
> is kibana running?
```

**Show Claude:**
- Detects the running instance from the previous session
- Reports URLs without restarting

---

## Part 4: Architecture highlights

### What makes this work for agents

| Feature | How |
|---------|-----|
| No interactive prompts | `[ -t 0 ]` detection, auto-accept |
| Structured status | `status.json` + `kbn-ctl status --json` |
| Parallel startup | ES clusters, Kibana waits run concurrently |
| Partial success | One instance failing doesn't tear down the other |
| Auto-retry | ES Serverless retries 3x for flaky uiam |
| Port cleanup | Kills orphaned processes before starting |
| Docker cleanup | Removes stale kibana-ci containers and dangling images |
| Chrome on macOS | `open -na` for separate profiles |
| Process cleanup | Kills by port + PID tree + Docker containers |

### File layout

```
commands/
├── bin/
│   ├── kbn              # startup orchestrator (870 lines)
│   ├── kbn-ctl          # control plane CLI (310 lines)
│   └── README.md        # human documentation
└── .claude/
    └── skills/
        ├── kibana-controller/
        │   ├── SKILL.md          # main skill (start, monitor, browser, auth)
        │   ├── failure-modes.md  # error patterns and fixes
        │   └── SKILL_TODO.md     # roadmap (hooks, subagent, etc.)
        └── kbn-status/
            └── SKILL.md          # /kbn-status quick check
```

---

## Talking points

- **Before**: multiple terminals, manual `yarn es`, `yarn start`, tracking PIDs, killing orphans
- **After**: one command for humans, one skill for Claude
- **Shareable**: no hardcoded paths, symlink to install, works on any machine
- **Resilient**: retries, partial success, port cleanup, Docker cleanup
- **Agent-native**: non-interactive mode, JSON status, curl auth patterns, browser login guidance

---

## Why this exists — native scripts vs. kbn

### The question

Kibana ships with `yarn es`, `yarn start`, `yarn serverless-es`, config
files for stateful (`kibana.stack.dev.yml`) and serverless
(`kibana.serverless.dev.yml`), and `--no-optimizer` flags. Why not just
let Claude run the native scripts directly?

### What you'd need to do natively (5 terminals, manual coordination)

To run both serverless and stateful side-by-side, you need to manually:

```bash
# Terminal 1: ES Serverless (Docker)
yarn es serverless --projectType elasticsearch_general_purpose --clean --kill

# Terminal 2: ES Stateful (snapshot, different ports)
yarn es snapshot --license trial --clean -E http.port=9201 -E transport.port=9301

# Terminal 3: Shared optimizer (otherwise each Kibana runs its own, ~8-16GB RAM)
node scripts/build_kibana_platform_plugins --watch

# Wait for optimizer to finish building...

# Terminal 4: Kibana Serverless (port 5601, no optimizer)
yarn serverless-es --server.port=5601 --no-optimizer

# Terminal 5: Kibana Stateful (port 5611, different cookie, no optimizer)
yarn start --config config/kibana.stack.dev.yml \
  --server.port=5611 \
  --xpack.security.cookieName=sid-stack \
  --no-optimizer
```

Plus: wait for each ES cluster to be ready before starting its Kibana,
run `node scripts/eis.js` for EIS after each cluster, set up vault,
manage Chrome profiles for independent logins, and clean up Docker
containers / orphaned node processes from previous failed runs.

### What kbn solves that native scripts don't

| Problem | Native scripts | kbn |
|---------|---------------|-----|
| **Dual-mode (SLS + Stack)** | Manual 5-terminal setup | One command |
| **Shared optimizer** | Each Kibana runs its own (~8-16GB total) | Single optimizer in watch mode, both Kibana use `--no-optimizer` |
| **Startup ordering** | Manually watch for "cluster ready" | Automatic: ES ready → EIS → Kibana |
| **Auth cookie conflict** | Must remember `--xpack.security.cookieName=sid-stack` | Built in |
| **Port conflicts** | Must remember port flags for ES (9201/9301) and Kibana (5611) | Built in |
| **EIS / vault** | Manual vault login + `node scripts/eis.js` per cluster | Pre-check, auto-retry, fallback to no-EIS |
| **Docker cleanup** | Stale uiam/es containers block next run | Auto-cleanup on start |
| **Port cleanup** | Orphaned node processes on 5601/5611 | Auto-kill before start |
| **Process lifecycle** | Ctrl+C one terminal, others keep running | Single Ctrl+C cleans everything |
| **Restart after code changes** | Kill processes, find PIDs, restart manually | `kbn-ctl restart kbnsls` |
| **Partial failure** | Serverless fails, you lose stateful too (manual) | Stateful stays up, serverless retries |
| **Status at a glance** | `lsof`, `docker ps`, check each terminal | `kbn-ctl status` |
| **Agent-friendly** | Interactive prompts, no structured output | Non-interactive mode, JSON status, `kbn-ctl` CLI |

### What could be simplified

- **Bootstrap on every start**: Currently runs `yarn kbn bootstrap` every
  time. Could be `--clean` only since devs already run it manually. But
  it's a no-op when nothing changed (~2s with cache).
- **Chrome profiles**: Nice-to-have but not essential. Most devs could
  just use incognito. Kept because agents need programmatic auth per
  instance.
- **tmux viewer**: Convenient but optional (`--quiet` skips it). Devs
  with their own terminal setup can ignore it.

### What an agent would need without kbn

Without `kbn`, the Claude skill would need to:

1. Detect nvm, switch node version, verify it stuck
2. Run 5 commands in correct order with correct flags
3. Know the port assignments (9200 vs 9201, 5601 vs 5611)
4. Know about the cookie name conflict
5. Know to run optimizer separately and use `--no-optimizer`
6. Watch log files for readiness patterns before starting Kibana
7. Handle vault login for EIS
8. Clean up stale Docker containers
9. Kill orphaned processes on ports
10. Manage all PIDs for cleanup
11. Handle partial failures independently

That's the entire `kbn` script reimplemented as skill instructions. The
skill would be 3-4x larger and more fragile. `kbn` + `kbn-ctl` abstract
the orchestration so the skill only needs to know: start, status, logs,
restart, stop.

### Verdict

**Not bloat.** The native scripts handle individual components. `kbn`
handles the composition — running both modes side-by-side with a shared
optimizer, coordinated startup, and clean lifecycle management. This is
the part that's genuinely hard to do manually and even harder to teach
an agent to do ad-hoc.
