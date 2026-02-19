_created_at: 2026-02-18_

# Prompt Standards

This file defines the standards that prompts in this project should conform to. Agents should read this before writing or evaluating any prompt.

---

## Prompt Contract Framework

_created_at: 2026-02-18_

Every non-trivial prompt should be structured as a **Prompt Contract** with four components. This replaces vague, open-ended instructions with a testable specification.

### 1. GOAL
Define the exact success metric — what *done* looks like.
- State the specific outcome, not a general direction.
- Make it testable: the result should be verifiable in under a minute.
- Include success criteria where relevant (e.g., a user can complete X action within Y seconds).

**Example:**
```
GOAL: Implement Stripe subscription management where users can subscribe to 3 tiers
(free/pro/team), upgrade/downgrade instantly, and see billing status on /settings/billing.
Success = a free user can subscribe to Pro, see the charge on Stripe dashboard,
and access gated features within 5 seconds.
```

### 2. CONSTRAINTS
Hard boundaries that cannot be crossed — stack, patterns, rules.
- Specify libraries, frameworks, and architecture that are non-negotiable.
- State what must not be changed, installed, or introduced.
- Include any schema, environment, or integration boundaries.

**Example:**
```
CONSTRAINTS: Convex useQuery for data, no polling, no SWR.
Clerk useUser() for auth check. Redirect to /sign-in if unauthenticated.
Max 150 lines per component file.
```

### 3. FORMAT
The specific structure of the expected output.
- Name the files and locations the output should be written to.
- Specify types, exports, return shapes, or documentation requirements.
- Define component boundaries explicitly (e.g., server vs. client components).

**Example:**
```
FORMAT:
1. Convex function in convex/users.ts (mutation, not action)
2. Zod schema in convex/schemas/onboarding.ts
3. TypeScript types exported from convex/types/user.ts
4. JSDoc on the public function
5. Return { success: boolean, userId: string, error?: string }
```

### 4. FAILURE CONDITIONS
Explicit criteria that make the output unacceptable.
- Define what "bad" looks like so the agent doesn't have to infer what "good" means.
- Cover wrong library choices, missing states, structural violations, and anti-patterns.
- Be specific — vague failure conditions are as useless as no failure conditions.

**Example:**
```
FAILURE CONDITIONS:
- Uses useState for data that should be in Convex
- Any component exceeds 150 lines
- Fetches data client-side when it could be server-side
- Uses any UI library besides Tailwind utility classes
- Missing loading and error states
- Missing TypeScript types on any function parameter
```

---

## Session Handshake

_created_at: 2026-02-18_

At the start of every session in a project that has a `CLAUDE.md` or `AGENTS.md`, the first message should instruct the agent to read and confirm its understanding of those constraints before doing any work:

```
Read CLAUDE.md and confirm you understand the project constraints before doing anything.
```

This creates an explicit acknowledgment that constraints are active. Without it, the agent may proceed without internalizing the project rules.

---

## Baseline Principles

_created_at: 2026-02-18_

When specific standards above are not yet defined, apply these defaults:

- **Clear intent** — The prompt states its goal without ambiguity.
- **Appropriate scope** — The prompt does not ask for more than is needed in a single call.
- **Consistent voice** — Tone and style are uniform across related prompts.
- **No unnecessary verbosity** — Instructions are tight; every sentence earns its place.
- **Explicit constraints** — Any hard rules (format, length, tone, off-limits content) are stated, not implied.

---

## Agent Behavior

_created_at: 2026-02-18_

When working in a project that involves prompts, agents must:

1. **Check for compliance** — Before using or generating prompts, evaluate whether the current context provides enough information to meet these standards.

2. **Ask before assuming** — If the necessary context is missing, ask the user:
   > "I have access to prompt standards defined in your global config. Would you like me to apply them to this session?"

3. **If the user says yes**, proceed step-by-step:
   - Ask one focused question at a time to gather missing context.
   - Do not front-load all questions at once; let each answer inform whether the next question is needed.
   - Once sufficient context is gathered, confirm the approach before writing or modifying prompts.
   - Apply the standards consistently for the remainder of the session.

4. **If the user says no**, proceed without enforcing these standards, but do not discard them — they may become relevant later.
