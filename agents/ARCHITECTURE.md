_created_at: 2026-06-17_

# Agent Infrastructure Architecture

Full-stack reference for building Claude-powered agents with semantic skill discovery, persistent memory, and external integrations. Covers two primary workflow patterns: **coding** (Claude Code / developer-facing) and **production** (automated / event-driven).

---

## System Overview

```mermaid
flowchart TB
    subgraph TRIGGERS["① Triggers"]
        DEV["Developer<br/>Claude Code CLI"]
        CRON["Cron / Schedule"]
        WEBHOOK["Webhook / Event Stream"]
    end

    subgraph ORCH["② Orchestration"]
        CCWF["Claude Code<br/>Workflow Engine<br/>(coding flows)"]
        LG["LangGraph<br/>State Machine<br/>(production flows)"]
    end

    subgraph CTX["③ Context &amp; Memory"]
        SR["<b>Skill Router MCP</b><br/>LlamaIndex ToolRetriever<br/>top-k skills"]
        C7["<b>Context7 MCP</b><br/>latest library docs"]
        BEADS["<b>Beads (bd)</b><br/>task state"]
        MEM0["<b>Mem0</b><br/>cross-session facts"]
        LETTA["<b>Letta</b><br/>context paging"]
        VEC["<b>Vector Store</b><br/>Chroma / pgvector"]
    end

    subgraph INF["④ Inference · Claude API"]
        OPUS["Opus 4.8<br/>complex planning"]
        SONNET["Sonnet 4.6<br/>balanced default"]
        HAIKU["Haiku 4.5<br/>fast / cheap"]
    end

    subgraph INT["⑤ Integrations (MCP)"]
        SLACK["Slack"]
        NOTION["Notion"]
        GMAIL["Gmail"]
        GITHUB["GitHub"]
        GCAL["Google Calendar"]
        LINEAR["Linear / Jira"]
        CUSTOM["Custom REST APIs"]
    end

    subgraph PERSIST["⑥ Persistence"]
        MEMDB["Memory DB<br/>(Mem0 / Letta store)"]
        TASKDB["Task DB<br/>(Beads .beads/)"]
        ARTIFACTS["Code / File Artifacts"]
        AUDIT["Audit Log<br/>(LangGraph checkpoints)"]
    end

    TRIGGERS --> ORCH
    ORCH -->|"query skills / docs / task state"| CTX
    CTX -->|"assembled context injected"| INF
    INF -->|"tool calls"| INT
    INF -.->|"write back learnings + task updates"| CTX
    CTX --> PERSIST
    ORCH --> PERSIST

    MEM0 <-->|"embed / retrieve"| VEC
    LETTA <-->|"embed / retrieve"| VEC
```

---

## Workflow A — Coding (Claude Code / Developer-Facing)

Each developer turn assembles context from three sources before inference: skills, memory, and docs. The agent never sees the full skill pool — only what the router returns.

```mermaid
sequenceDiagram
    actor Dev as Developer
    participant CC as Claude Code
    participant SR as Skill Router MCP<br/>(LlamaIndex)
    participant MEM as Mem0
    participant C7 as Context7 MCP
    participant BD as Beads
    participant CL as Claude API<br/>(Sonnet 4.6)
    participant INT as Integration MCPs

    Dev->>CC: types request

    par Context assembly (parallel)
        CC->>SR: search_skills(query)
        SR-->>CC: top-3 skill names + descriptions
        CC->>MEM: recall(query, user_id)
        MEM-->>CC: relevant past facts / patterns
        CC->>C7: resolve_docs(libs_in_scope)
        C7-->>CC: current API signatures
        CC->>BD: get_active_tasks()
        BD-->>CC: current task state
    end

    CC->>CL: assembled prompt<br/>[skills + memory + docs + task state]

    loop Agent turns
        CL->>INT: tool calls (GitHub, Notion, Slack…)
        INT-->>CL: results
        CL->>BD: update_task(step, status)
    end

    CL-->>CC: final response + artifacts

    par Post-turn writes (parallel)
        CC->>MEM: add(new facts, patterns observed)
        CC->>BD: complete_task() or update_task()
    end

    CC-->>Dev: response
```

### Key decisions in coding flows

| Decision | Choice | Why |
|---|---|---|
| Model for most tasks | Sonnet 4.6 | Balanced cost/quality for tool-heavy coding turns |
| Model for hard planning | Opus 4.8 | Architecture decisions, ambiguous requirements |
| Model for quick checks | Haiku 4.5 | Lint feedback, classification, fast validation |
| Skill discovery | LlamaIndex ToolRetriever via MCP | Never loads all 200 skills; top-k injected on demand |
| Memory | Mem0 | Extracts facts automatically; no manual annotation |
| Context window management | Handoff pattern from `learnings.md` | At ~75% usage, emit structured HandoffPayload; orchestrator spins new agent |

---

## Workflow B — Production System (Automated / Event-Driven)

LangGraph owns the state machine. Each node is a discrete Claude call; checkpoints make the flow resumable. Mem0 provides history that spans multiple past runs.

```mermaid
flowchart TD
    TRIG["<b>Trigger</b><br/>cron · webhook · queue"]

    subgraph LG_GRAPH["LangGraph State Machine"]
        INIT["<b>init_node</b> · Haiku<br/>load event payload<br/>fetch Mem0 + Beads context"]
        PLAN["<b>plan_node</b> · Opus 4.8<br/>break work into steps<br/>select tools needed"]
        EXEC["<b>exec_node ×N</b> · Sonnet 4.6<br/>execute one step<br/>call MCP integrations"]
        VERIFY["<b>verify_node</b> · Haiku 4.5<br/>check output quality<br/>detect errors / retries"]
        NOTIFY["<b>notify_node</b> · Haiku 4.5<br/>send summary<br/>update tickets"]
        END_NODE["<b>end_node</b><br/>write Mem0 learnings<br/>close Beads · emit audit log"]
    end

    subgraph INFRA["Supporting Infrastructure"]
        MEM0B["<b>Mem0</b><br/>historical run context"]
        BEADSB["<b>Beads</b><br/>task tracking"]
        CHKPT["<b>LangGraph checkpoints</b><br/>(resumable)"]
    end

    subgraph MCPINT["MCP Integrations"]
        S["Slack"]
        N["Notion"]
        G["Gmail"]
        GH["GitHub"]
        GC["Google Calendar"]
        LIN["Linear"]
        API["Custom APIs"]
    end

    TRIG --> INIT
    INIT -->|"context loaded"| PLAN
    PLAN -->|"step list"| EXEC
    EXEC -->|"step result"| VERIFY
    VERIFY -->|"pass"| EXEC
    VERIFY -->|"all steps done"| NOTIFY
    VERIFY -->|"fail > max retries"| NOTIFY
    NOTIFY --> END_NODE

    LG_GRAPH <-->|"recall / write context"| INFRA
    EXEC <-->|"tool calls"| MCPINT
```

### Context window handoff in production flows

LangGraph checkpoints handle resumability at the *flow* level, but individual nodes still hit context limits. Apply the handoff pattern from `learnings.md`:

```
plan_node emits → PlanPayload { steps[], tools[], constraints[] }
exec_node(n) emits → StepPayload { completedSteps[], pendingSteps[], artifacts{}, criticalContext{} }
verify_node receives StepPayload directly — never the full history
```

This keeps each node's input bounded regardless of how long the overall run is.

---

## Library Stack Summary

```
┌─────────────────────────────────────────────────────────────────┐
│  Inference                                                      │
│  Anthropic SDK  ·  claude-opus-4-8 / sonnet-4-6 / haiku-4-5    │
├─────────────────────────────────────────────────────────────────┤
│  Orchestration                                                  │
│  Claude Code Workflow Engine  (coding flows)                    │
│  LangGraph  (production flows, stateful, resumable)             │
├─────────────────────────────────────────────────────────────────┤
│  Skill / Tool Discovery                                         │
│  LlamaIndex ToolRetrieverMixin  ←→  Skill Router MCP server    │
│  Indexes: skill name + description + trigger patterns           │
│  Returns: top-k relevant skills, loaded into context on demand  │
├─────────────────────────────────────────────────────────────────┤
│  Memory                                                         │
│  Mem0  (cross-session, automatic fact extraction)               │
│  Letta  (long-running agents, virtual context paging)           │
│  Vector backend: Chroma (local dev)  ·  pgvector (production)  │
├─────────────────────────────────────────────────────────────────┤
│  Task State                                                     │
│  Beads (bd)  ←  source of truth for in-progress work           │
├─────────────────────────────────────────────────────────────────┤
│  Docs / Reference                                               │
│  Context7 MCP  ←  always-current library docs                  │
├─────────────────────────────────────────────────────────────────┤
│  External Integrations  (all via MCP servers)                  │
│  Slack · Notion · Gmail · GitHub · Google Calendar             │
│  Linear · Custom REST APIs                                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## Build Order

Start here if building from scratch. Each layer depends on the one below it.

1. **Anthropic SDK** — wire up Claude API, confirm model calls work
2. **Beads** — `bd init` in each project; task state before anything else
3. **Mem0** — add memory writes after each agent turn; reads at prompt assembly
4. **Skill Router MCP** — index existing skills with LlamaIndex; expose `search_skills` as MCP tool
5. **Context7** — configure MCP server; inject doc snippets for libs in scope
6. **LangGraph** — wrap production flows in state machines; add checkpointing
7. **MCP Integrations** — add external tools one at a time as needed

Resist adding all layers at once. The value compounds: you need memory before you can benefit from skill routing, and you need both before LangGraph orchestration pays off.

---

## Files in This Directory

| File | Description |
|---|---|
| `ARCHITECTURE.md` | This file — full infrastructure diagram and workflow patterns |
| `learnings.md` | Hard-won context management patterns (handoff, summarization, paging) |
| `extensions/AGENTS.md` | Which tools, MCPs, and skills agents should use |
| `prompts/STANDARDS.md` | Prompt quality standards for reusable prompt files |
| `subagents/usemybrain.md` | Subagent configuration |
| `skills/` | Local skill definitions |
