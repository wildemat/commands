---
name: prepare-contract
description: gather information from a user on their project, scope, reference material, examples, requirements. Then take that and use it to create a prompt contract according to the format below. Prompt contract format can be followed for any size or complexity task. Prompt contract is meant to provide a clear outline for the agent in a new working session.
---

## Prompt Contract Framework

_created_at: 2026-02-18_

Every non-trivial prompt should be structured as a **Prompt Contract** with four components. This replaces vague, open-ended instructions with a testable specification.

### 1. GOAL

Define the exact success metric — what _done_ looks like.

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

### 5. SUPPORTING RESOURCES

Context for the next agent session to better complete the task according to the contract.
This may include relevant material originally provided by the user to build the contract, as well as extra
content provided by the agent to further improve scope of the contract.

**Example:**

```
SUPPORTING RESOURCES:
- Helper function at /src/server/util/helper.ts
- Official documentation at http://site.com/api/docs/getData
- Previous commit with a similar structure at github.com/repo/commit/asdf
- Architecture diagram generated at: {filepath}
```
