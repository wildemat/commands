# Kibana Controller — Failure Modes

Reference for diagnosing and fixing common kbn-dev startup and runtime failures.

## Startup failures

### All processes die immediately

**Symptom:** `kbn-dev-ctl status` shows everything down, logs are nearly empty.

**Likely cause:** Wrong Node.js version, or `pnpm` missing from PATH. Agent
shells don't load nvm, so the system node doesn't match kibana's `.nvmrc` — and
since pnpm is installed per node version, losing nvm loses pnpm too.

**Fix:**
```bash
# Source nvm first (required for all pnpm commands in agent shells)
source "${NVM_DIR:-$HOME/.nvm}/nvm.sh" --no-use && nvm use --silent
node --version && pnpm --version
kbn-dev-ctl logs main --grep "incompatible\|Expected version\|pnpm not found"
# If node mismatch: nvm install $(cat .nvmrc)
# If pnpm missing:  corepack enable pnpm && corepack prepare pnpm@11.21.0 --activate
```

### pnpm missing or wrong version

**Symptom:** `kbn-dev` exits immediately with `ERROR: pnpm not found`, or warns
that the installed pnpm doesn't satisfy `engines.pnpm`.

**Cause:** Kibana pins pnpm via `package.json` `engines.pnpm` (currently
`~11.21.0`), deliberately *without* a `packageManager` field so yarn classic can
still run scripts. Corepack therefore won't auto-provision pnpm — it must
already be on PATH, and it is installed per node version.

**Fix:**
```bash
corepack enable pnpm
corepack prepare pnpm@$(node -p "require('./package.json').engines.pnpm.replace(/^[~^]/,'')") --activate
# or, if you use mise: mise install   (.mise.toml pins the same version)
```

### ES clusters fail to start

**Symptom:** `essls` or `esstack` shows `"alive": false` within seconds.

**Likely causes:**
1. Docker not running — `docker ps` fails
2. Ports 9200/9300 occupied — OrbStack, another ES instance, etc.
3. Cache corruption — stale `.es/cache` directory
4. Stale kibana-ci containers — `uiam` or ES containers from a previous run

**Fix:**
```bash
docker ps                                    # verify Docker is running
kbn-dev-ctl logs essls --grep "error"   # check specific error
# Restart with clean: kbn-dev --clean
```

### uiam container fails to start (serverless)

**Symptom:** Log shows `Waiting for "uiam" container (unhealthy)` then
`The "uiam" container failed to start within the expected time`.

**Cause:** Stale Docker network or containers from a previous run. The
`elastic` Docker network accumulates stale DNS entries from crashed
containers, causing uiam to fail resolving cosmosdb. `kbn-dev` cleans
containers and the network automatically on startup, and retries up to
3 times.

**Fix if it persists:**
```bash
kbn-dev-ctl stop
docker system prune -a    # nuclear option: clears all Docker state
kbn-dev
```

### Bootstrap fails

**Symptom:** Status stuck at `"state": "starting"`, optimizer never starts.

**Likely causes:**
1. Missing dependencies after branch switch
2. Corrupted `node_modules`
3. Lockfile drift — `pnpm-lock.yaml` doesn't match the manifests
4. A dependency with install scripts is not listed in `pnpm-workspace.yaml`
   `allowBuilds` (`strictDepBuilds: true` makes this fatal)

**Fix:**
```bash
kbn-dev-ctl logs main --tail 100        # check bootstrap output
kbn-dev-ctl stop

# Level 1 — caches and build output only:
pnpm run kbn clean && pnpm run kbn bootstrap

# Level 2 — node_modules is suspect:
pnpm run kbn bootstrap --force-install

# Level 3 — nuke node_modules, the pnpm store, and yarn-era leftovers:
pnpm run kbn reset && pnpm run kbn bootstrap

kbn-dev --quiet &
```

`kbn clean` no longer touches `node_modules` — that's what `--force-install`
and `kbn reset` are for.

### Switching between pre-pnpm and post-pnpm branches

**Symptom:** After checking out an older branch (or returning from one),
bootstrap fails or Kibana starts with `Cannot find module '@kbn/...'`.

**Cause:** Kibana migrated Yarn → pnpm in
[#284611](https://github.com/elastic/kibana/pull/284611). Branches on either
side of that produce incompatible `node_modules` layouts, and the yarn-era
caches (`.yarn`, `.yarn-local-mirror`) linger.

Check which package manager produced the current tree:
```bash
ls node_modules/.modules.yaml >/dev/null 2>&1 && echo pnpm || echo "yarn (or missing)"
```

**Fix:** `pnpm run kbn reset && pnpm run kbn bootstrap` — `reset` clears
`node_modules`, `.pnpm-store`, and the yarn-era caches in one pass.

### Optimizer build succeeds but startup never advances

**Symptom:** `state` sits at `es_starting` indefinitely. `optimizer` shows
`"alive": true`, `kbnsls`/`kbnstack` never start, and `optimizer.log` ends with
a *successful* build line that stopped updating minutes ago.

Distinguish from "Optimizer dies" below by the `alive` flag: here the process is
healthy and the build worked — `kbn-dev` is stuck in `wait_for_log` on a ready
pattern the running optimizer never prints.

**Cause:** engine/pattern mismatch. Kibana's optimizer entrypoints have moved
over time: `scripts/build_rspack_bundles.js` existed alongside a genuinely
webpack `scripts/build_kibana_platform_plugins.js`, then the former was dropped
and the latter became an `@kbn/rspack-optimizer` shim. Selecting the engine by
which file exists gets this wrong on current branches and pairs the rspack
script with the webpack ready pattern
(`succ ... bundles compiled successfully`), which rspack never emits — it prints
`succ RSPack build completed in Ns`.

**Check:**
```bash
tail -5 ~/.kbn-dev/logs/optimizer.log            # which engine actually ran?
grep -l "@kbn/rspack-optimizer" \
  "$KBN_DIR/scripts/build_kibana_platform_plugins.js"   # rspack shim?
```

`resolve_optimizer` in `kbn_dev.sh` greps that shim to pick the engine, so
script and ready pattern always agree. If a future entrypoint rename
reintroduces the hang, update `resolve_optimizer` — not the call sites.

**Side effect:** when the wrong branch is taken, `kill_port 5678` is skipped and
rspack's HMR watcher from the previous run survives `kbn-dev-ctl stop`
(reparented to init). Check for strays after any optimizer hang:
```bash
pgrep -fl "build_kibana_platform_plugins|build_rspack_bundles"
```

### Optimizer dies

**Symptom:** `optimizer` shows `"alive": false`, Kibana processes never start.

**Likely cause:** Build errors in plugin code, out-of-memory.

**Fix:**
```bash
kbn-dev-ctl logs optimizer --tail 100
# Usually requires fixing the code error, then:
kbn-dev-ctl stop
kbn-dev --quiet &
```

## Runtime failures

### Kibana crashes after branch switch

**Symptom:** `FATAL: Cannot find module '@kbn/...'`

**Fix:**
```bash
kbn-dev-ctl stop
kbn-dev --quiet --clean &
```

### Port already in use

**Symptom:** `FATAL Error: Port 5601 is already in use`

**Fix:**
```bash
kbn-dev-ctl restart kbnsls   # or kbnstack for port 5611
```

The restart command kills the orphaned process and the monitor loop
restarts Kibana automatically.

### Kibana running but not responding

**Symptom:** Port is open but `curl` times out or returns 503.

**Check:**
```bash
kbn-dev-ctl logs kbnsls --tail 20 --grep "status\|ERROR\|FATAL"
curl -s -o /dev/null -w "%{http_code}" http://localhost:5601/api/status
```

If the status API returns 503, Kibana is still initializing plugins.
Wait and re-check. If it persists, restart.

### EIS / Vault failures

**Symptom:** Vault access fails in non-interactive mode, or `kbn-dev-ccm`
errors in the main log.

**Fix:**
```bash
# Option 1: disable EIS entirely
KBN_INFERENCE_URL="" kbn-dev --quiet &

# Option 2: provide API key directly
export KIBANA_EIS_CCM_API_KEY="your-key-here"
kbn-dev --quiet &

# Option 3: log in to vault first (interactive terminal)
VAULT_ADDR=https://secrets.elastic.co:8200 vault login -method oidc
```

## Log patterns to grep

| Pattern | Meaning |
|---------|---------|
| `succ Serverless ES cluster running` | ES Serverless ready |
| `succ ES cluster is ready` | ES Stateful ready |
| `succ.*bundles compiled successfully` | Optimizer build done |
| `succ all bundles cached` | Optimizer using cache |
| `[INFO ][status] Kibana is now available` | Kibana accepting requests |
| `FATAL` | Unrecoverable error |
| `server crashed` | Kibana process died |
| `Port .* is already in use` | Orphaned process on port |
| `Cannot find module` | Missing dependency (needs bootstrap) |
| `ERR_PNPM_OUTDATED_LOCKFILE` | `pnpm-lock.yaml` out of sync with manifests |
| `ERR_PNPM_NO_MATCHING_VERSION` | Lockfile references a version no longer published |
| `Use pnpm instead` | An install ran under yarn/npm (`preinstall_check.js`) |
