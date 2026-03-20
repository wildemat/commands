## Context saving tips

- Always pipe command outputs to a file and have agent grep for what it needs instead of reading in stdio
- When asked to copy files, should use cli and not read the whole file and then write the contents
- Specify context window to agents and orchestrate handoff

### Claude suggestion:

1. Track tokens externally, inject stop signal
   Count tokens on each response (via usage.input_tokens + usage.output_tokens from the API response), and when you hit a threshold, inject a system/user message like:

"You are approaching context limit. Summarize your current state, completed work, remaining tasks, and any critical context needed for handoff. Output this as structured JSON."

Then pass that summary as the initial context for the next agent.

2. Prompt the agent to self-report
   In the system prompt, instruct the agent to periodically output a structured status block (e.g. every N steps) including progress and what's left. You parse that and decide when to cut over.

3. Hard truncation with summarization
   Keep a rolling message history. When input_tokens approaches your threshold (e.g. 80% of context window), run a separate summarization call on the history, replace the history with the summary, and continue — either in the same agent or a new one.
   Handoff pattern that works well:
   Agent 1 → hits threshold → emits HandoffPayload { completedSteps, pendingTasks, artifacts, criticalContext }
   Orchestrator → receives payload → spins up Agent 2 with payload as initial context
   Enforcement reality check: You can't force an agent mid-generation to stop — you control the loop between turns. So enforcement happens at the orchestrator level between API calls, not within a single streaming response.
   The most robust approach is #1 + #3 combined: track usage on every response, trigger a structured handoff prompt at ~75% context, and pass a compressed summary to the next agent.
