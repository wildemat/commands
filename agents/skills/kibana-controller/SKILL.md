---
name: kibana-controller
description: >
  Start, stop, monitor, and interact with local Kibana dev instances
  (serverless on :5601, stateful on :5611) and their Elasticsearch clusters.
  Use when the user asks about Kibana status, needs to start or restart
  Kibana, is debugging startup failures, after code changes that require
  a Kibana restart, or when working in the kibana repo and the dev
  environment needs managing. Also use when the user says "kbn", "kibana
  dev", "start kibana", "restart kibana", "check kibana", "kibana logs",
  "es logs", or "kibana status".
allowed-tools: >
  Bash(kbn-ctl *)
  Bash(kbn *)
  Bash(curl *)
  Bash(lsof *)
  Bash(kill *)
  Bash(tail *)
  Bash(grep *)
---

# Kibana Controller

Manage a local Kibana development environment running two Kibana instances
(serverless + stateful) against two Elasticsearch clusters, with a shared
optimizer in watch mode.

## Architecture

```
kbn (orchestrator, long-running)
├── ES Serverless  (Docker, port 9200)
├── ES Stateful    (Docker, port 9201)
├── Optimizer      (watch mode, shared)
├── Kibana SLS     (port 5601, serverless mode)
└── Kibana Stack   (port 5611, stateful mode)
```

`kbn` is the startup orchestrator. `kbn-ctl` is the control plane.

## Locating the scripts

```
!`echo "kbn:     $(which kbn 2>/dev/null || echo 'NOT FOUND — install with: ln -sf /path/to/commands/bin/kbn /usr/local/bin/kbn')" && echo "kbn-ctl: $(which kbn-ctl 2>/dev/null || echo 'NOT FOUND — install with: ln -sf /path/to/commands/bin/kbn-ctl /usr/local/bin/kbn-ctl')"` 
```

If the scripts are not found, ask the user to install them per the README
in the commands repo, or locate them with `find ~ -name kbn-ctl -type f`.

## Working directory check

```
!`if [ -f "package.json" ] && grep -q '"name": "kibana"' package.json 2>/dev/null; then echo "cwd_is_kibana=true"; elif pwd | grep -q "/kibana"; then echo "cwd_is_kibana_subdir=true dir=$(pwd | sed 's|/kibana/.*|/kibana|')"; else echo "cwd_is_kibana=false"; fi`
```

Before doing anything, check the output above:
- `cwd_is_kibana=true` — you're at the kibana root. Proceed.
- `cwd_is_kibana_subdir=true` — you're inside kibana but not at the root.
  Use the `dir` value for `cd` when running kbn commands.
- `cwd_is_kibana=false` — you're not in a kibana repo. Ask the user:
  "I need to be in the Kibana repo to manage the dev environment.
  Should I navigate there? If so, where is your kibana checkout?"
  Then `cd` to their answer before proceeding. Try common locations
  first: `~/workplace/kibana`, `~/kibana`, `~/dev/kibana`.

All `kbn` and `kbn-ctl` commands must run from the kibana repo root.

## Current status

Check status before taking any action:

```
!`kbn-ctl status --json 2>/dev/null || echo '{"running": false, "state": "not_running"}'`
```

## UX guidelines — be concise

**Check status first.** The dynamic injection above gives you the current
state. Handle each case:

- **Already running, both ready**: Tell the user "Kibana is already running"
  and show the URLs. Do NOT restart unless the user explicitly asks.
- **Already running, partially ready**: Report what's up and what's down.
  Offer to restart the failed component.
- **Running but not ready yet**: A previous session started kbn and it's
  still booting. Poll for readiness (step 3 below) instead of restarting.
- **Not running**: Start it (steps 1-5 below).

If the user says "restart kibana", use `kbn-ctl stop` then start fresh.

**Starting Kibana is a background task.** Don't show the user every poll
cycle. Follow this pattern:

1. Tell the user: "Spinning up serverless and stateful Kibana, standby... (run /kbn-status to check anytime)"
2. Run `kbn-ctl start --quiet` in the background (it finds the kibana
   repo automatically and checks if already running).
3. Poll silently with a **simple sleep loop** — do NOT show output:
   ```bash
   for i in $(seq 1 40); do
     sleep 15
     kbn_state=$(kbn-ctl status --json 2>/dev/null)
     sls=$(echo "$kbn_state" | grep -c '"kbnsls".*"ready": true')
     stack=$(echo "$kbn_state" | grep -c '"kbnstack".*"ready": true')
     is_running=$(echo "$kbn_state" | grep -c '"running": true')
     if [ "$sls" -gt 0 ] && [ "$stack" -gt 0 ]; then break; fi
     if [ "$is_running" = "0" ] && [ $i -gt 2 ]; then break; fi
   done
   ```
   Important: check `"running": true` — if the orchestrator died (`false`)
   after a few polls, stop waiting and report the failure.
   The `state` field goes: `starting` → `es_starting` → `optimizer_ready`
   → `running`. There is NO `"ready"` state.
4. Check what came up and tell the user:
   - Both ready: "Kibana is ready! Serverless: :5601, Stateful: :5611"
   - Neither ready: "Startup failed. Run `kbn-ctl logs essls --grep ERROR`
     to see what went wrong."
   - **Stateful up, Serverless failed**: Tell the user Stateful is ready,
     then automatically attempt to recover Serverless (see below).
5. If the `kbn` background process exited early (exit code 0), check
   `kbn-ctl status --json` — if `"state": "stopped"` or `"state": "failed"`,
   the orchestrator tore itself down. Check `kbn-ctl logs main --tail 20`
   for the cause.

### Auto-recover Serverless when Stateful is up

If Stateful comes up but Serverless fails (usually uiam container timeout),
automatically retry without user intervention:

1. Tell the user: "Stateful is ready at :5611. Serverless failed — retrying..."
2. Clean up stale serverless containers:
   ```bash
   docker rm -f $(docker ps -aq --filter "name=es01" --filter "name=es02" --filter "name=es03" --filter "name=uiam" --filter "name=uiam-cosmosdb") 2>/dev/null
   ```
3. Restart the serverless ES cluster from the kibana repo:
   ```bash
   cd /path/to/kibana && yarn es serverless --projectType elasticsearch_general_purpose --clean --kill &
   ```
4. Poll `kbn-ctl status --json` for `essls.ready` becoming true (up to 5 min).
5. Once ES Serverless is ready, start Kibana Serverless:
   ```bash
   cd /path/to/kibana && KBN_OPTIMIZER_USE_MAX_AVAILABLE_RESOURCES=false yarn serverless-es --server.port=5601 --no-optimizer &
   ```
6. Poll for `kbnsls.ready` or HTTP 200 on port 5601 (up to 2 min).
7. Report result: "Serverless is now ready at :5601" or "Serverless
   recovery failed — uiam may be broken. Try `kbn --quiet --clean`."

Retry this recovery once. If it fails twice, stop and inform the user.

**Never** show raw JSON status output unless the user explicitly asks.
**Never** show intermediate polling results. One message at the start,
one message when ready (or failed).

## Commands

| Action | Command |
|--------|---------|
| Start | `kbn-ctl start --quiet` |
| Start clean | `kbn-ctl start --quiet --clean` |
| Start (no EIS) | `KBN_INFERENCE_URL="" kbn-ctl start --quiet` |
| Status (JSON) | `kbn-ctl status --json` |
| Status (human) | `kbn-ctl status` |
| Logs | `kbn-ctl logs <component> [--tail N] [--grep PATTERN]` |
| Restart Kibana | `kbn-ctl restart <kbnsls\|kbnstack>` |
| Stop | `kbn-ctl stop` |

Components: `essls`, `esstack`, `optimizer`, `kbnsls`, `kbnstack`, `main`, `all`

## When to restart Kibana

Check `kbn-ctl status --json` and restart if a Kibana component shows
`"alive": false` — common after branch switches, plugin code changes,
or config edits. The monitor loop in `kbn` auto-restarts after the kill.
ES clusters and the optimizer stay running.

## Instance details and authentication

| Instance | URL | Login | ES port |
|----------|-----|-------|---------|
| Serverless | http://localhost:5601 | auto-login (dev mode) | 9200 |
| Stateful | http://localhost:5611 | elastic / changeme | 9201 |

### Curl with auth

**Stateful** uses basic auth:
```bash
curl -s -u elastic:changeme http://localhost:5611/api/status
curl -s -u elastic:changeme http://localhost:5611/app/elasticsearch/start
curl -s -u elastic:changeme "http://localhost:5611/api/console/proxy?path=_cat/indices&method=GET"
```

**Serverless** in dev mode uses cookie-based auth. First get a session
cookie, then use it:
```bash
# Get session cookie (serverless dev auto-creates a session)
curl -s -c /tmp/kbn-sls-cookie -L http://localhost:5601/app/home
# Use the cookie for subsequent requests
curl -s -b /tmp/kbn-sls-cookie http://localhost:5601/api/status
curl -s -b /tmp/kbn-sls-cookie http://localhost:5601/app/elasticsearch/start
```

**ES clusters directly:**
```bash
# Serverless ES (port 9200) — no auth in dev
curl -s http://localhost:9200/_cat/indices?v
# Stateful ES (port 9201) — basic auth
curl -s -u elastic:changeme http://localhost:9201/_cat/indices?v
```

When the user asks "is X working" and you don't have browser tools,
use curl to fetch the page and check for expected content, API status
codes, or error messages.

## Failure diagnosis

See [failure-modes.md](failure-modes.md) for detailed patterns. Key fixes:
- **Node mismatch**: `nvm install $(cat .nvmrc)`
- **Port in use**: `kbn-ctl restart <component>`
- **Docker not running**: start Docker
- **After branch switch**: `kbn --quiet --clean`
- **Vault failed**: `KBN_INFERENCE_URL="" kbn --quiet`

## Browser interaction

When the user asks about what's visible in Kibana, whether a feature is
working, or to check/test UI behavior:

1. **If browser tools are available** (e.g. `cursor-ide-browser` MCP),
   use them to navigate and visually inspect the running instances.
2. **If no browser tools**, use `curl` with the auth patterns above to
   fetch pages and check for expected content or error codes.

Do NOT search the codebase for UI questions that can be answered by
looking at or curling the running app.

- **Serverless** (port 5601): navigate to `http://localhost:5601`.
  If a login/role selection screen appears, select the **admin** role.
  The dev serverless setup uses mock authentication — no password needed,
  just pick the role from the selector.
- **Stateful** (port 5611): navigate to `http://localhost:5611`.
  If a login screen appears, enter username `elastic` and password
  `changeme`. Always use these credentials for stateful.

Common Kibana app paths:
- Getting started / onboarding: `/app/elasticsearch/start`
- Search / indices: `/app/enterprise_search/elasticsearch`
- Dev Tools console: `/app/dev_tools#/console`
- Discover: `/app/discover`
- Stack Management: `/app/management`
- Dashboards: `/app/dashboards`

When the user says "is X working on serverless" or "check the Y page",
use the browser to navigate there and verify visually.

## Proactive monitoring

After editing `.ts`, `.tsx`, `.yml`, or config files, silently run
`kbn-ctl status --json`. If a component is down, restart it and tell
the user briefly: "Kibana SLS crashed, restarting..."
