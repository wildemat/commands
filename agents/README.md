_created_at: 2026-02-18_

# Agent Configuration Home

This directory is the central home for agent configuration data, including prompts, settings, and other resources used to guide and customize AI agent behavior.

## Purpose

When an agent is pointed to this directory, it should treat the contents here as authoritative configuration and context for its operation. Read this README first, then use the source map below to navigate to files relevant to your current task.

---

## Source Map

### `/prompts/`
Reusable prompt files, templates, and prompt-quality standards.

| File | Description |
|------|-------------|
| `STANDARDS.md` | Standards that prompts should conform to. Agents working with prompts should read this first and apply the defined behavior when context is insufficient. |

---

## Instructions for Agents

1. Read this README first.
2. Use the source map above to locate files relevant to your task.
3. Use these global rules to instantiate an `AGENTS.md` file in the root of your current project directory:
   - Distill the constraints, guardrails, and conventions from this config that are relevant to the project at hand.
   - Write for a human audience: clear, concise, and scannable — not a raw dump of config data.
   - Include only what matters for the project; omit rules that have no bearing on it.
4. If an `AGENTS.md` already exists in the project root, do not overwrite it. Instead, create `AGENTS-LOCAL.md` in the same directory and add it to the session's context. Apply the same standards: relevant, human-readable, non-redundant with the existing file.
5. The resulting `AGENTS.md` or `AGENTS-LOCAL.md` should function as a quick-reference guardrail document — something a developer or another agent can read in under a minute and understand the key rules in play.

---

## Timestamping Convention

All files and sections in this directory must be timestamped. Agents must follow this convention when creating or editing any file here.

**When creating a new file:**
Add these lines at the very top of the file, before any other content:
```
_created_at: YYYY-MM-DD_
```

**When adding a new section to an existing file:**
Add a timestamp on the line immediately after the section heading:
```
## Section Title
_created_at: YYYY-MM-DD_
```

**When editing existing content:**
Add or append an `_edited_` line directly beneath the existing `_created_at_` line for that file or section:
```
_created_at: YYYY-MM-DD_
_edited: YYYY-MM-DD_
```

Only add an `_edited_` entry when the edit occurs on a **different day** than the `_created_at_` date. Edits on the same day as creation (or as the last edit) require no new entry. Multiple edits across different days each get their own line in chronological order.

---

## Keeping the Source Map Current

**Any agent that adds a file or directory to this folder must update the source map in this README.**

When adding an entry:
- Add the file or directory under the appropriate section, or create a new section if it introduces a new category.
- Write a single-sentence description that tells another agent what the file contains and when to use it.
- Keep descriptions brief — this is a navigation aid, not documentation.
- Do not remove or alter existing entries unless the file has been deleted or renamed.
- Follow the timestamping convention above when making edits to this file.
