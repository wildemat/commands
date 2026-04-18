# Kibana Controller — Roadmap

## Current state (Option B — instruction-based monitoring)

The skill instructs Claude to check `kbn-ctl status` after editing files
and proactively restart Kibana when needed. This relies on the model
following the instructions in SKILL.md.

## Next: Hook-based monitoring (Option A)

Use Claude Code's `PostToolUse` hooks to automatically run a health check
after every file edit. This makes monitoring deterministic — the hook fires
regardless of whether the model remembers to check.

### Implementation

Add to `.claude/settings.json` (or the kibana project's settings):

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "kbn-ctl status --json 2>/dev/null | jq -r 'if .running then ([.components | to_entries[] | select(.value.alive == false or (.value.ready == false and .value.ready != null)) | .key] | if length > 0 then \"⚠️  Unhealthy components: \" + join(\", \") else empty end) else empty end' 2>/dev/null"
          }
        ]
      }
    ]
  }
}
```

This runs after every Edit/Write. If all components are healthy, it produces
no output (Claude doesn't see it). If something is down, it emits a warning
that Claude sees and can act on.

### Considerations

- The hook runs on every file edit, including non-Kibana files. The `jq`
  pipeline is lightweight (~10ms) and produces no output when kbn
  isn't running, so the overhead is negligible.
- Could add `paths` filtering to the skill to only activate when editing
  `.ts`, `.tsx`, `.yml`, or config files.
- The hook needs `jq` installed. Could fall back to grep-based parsing.

## Future enhancements

### Background subagent for continuous monitoring

```yaml
---
name: kibana-monitor
description: Background monitor for Kibana dev environment health
background: true
tools: Bash
model: haiku
maxTurns: 50
---
```

A background subagent that polls `kbn-ctl status --json` every 30 seconds
and builds up a timeline of events. Returns a summary when done. Useful for
long coding sessions where you want a retrospective on stability.

**Blocked on:** Background subagents can't proactively interrupt the main
conversation. They return results only when complete. This limits their
utility as a real-time monitor. Revisit when Claude Code adds push
notifications from background subagents.

### kbn-ctl enhancements

- `kbn-ctl restart optimizer` — rebuild the optimizer without restarting ES
- `kbn-ctl bootstrap` — run bootstrap while keeping ES alive
- `kbn-ctl switch-branch` — orchestrate branch switch: stop Kibana, switch,
  bootstrap, restart Kibana (keep ES running)
- `kbn-ctl health` — deeper health check: curl each endpoint, verify ES
  cluster health API, check disk space

### Subagent for browser interaction

Create a subagent with the browser MCP that can:
- Navigate to specific Kibana apps via URL
- Verify the UI rendered correctly after code changes
- Take screenshots for visual regression
- Log in and perform smoke tests

```yaml
---
name: kibana-browser
description: Interact with running Kibana instances via browser
tools: Read, Bash
mcpServers:
  - cursor-ide-browser
---
```

### Distribution to the Kibana repo

Once stable, move:
- The skill to `kibana/.claude/skills/kibana-controller/`
- The subagent to `kibana/.claude/agents/kibana-monitor.md`
- `kbn` and `kbn-ctl` to `kibana/scripts/` or distribute via
  the team's shared tooling

This makes the entire setup available to all Kibana devs who use Claude Code.
