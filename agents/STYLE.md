_created_at: 2026-07-03_

# Writing Standards

Applies to all code and all agent responses. Read at the start of every session.

## Code Comments

Principle: **code says how, comments say why.** Write one only when the reason
isn't already visible in the code.

- **Default to none.** A comment earns its place by capturing what the code
  can't: a non-obvious constraint, a tradeoff, a rejected alternative, or a bug
  workaround (link the issue/PR).
- **Assume an expert reader.** Don't explain the language, framework, or
  codebase basics — in Kibana, mappings/plugins/lifecycles/DI are assumed
  knowledge. Litmus test: if a competent engineer in this area already knows it,
  cut it.
- **Signatures over bodies.** One line on an exported function, type, or public
  API — what it is and how it fits — is the highest-value comment. If a body
  needs narration, simplify the code instead.
- **No author-private context.** Omit "we decided," references to our
  conversations, issue numbers as justification, and asides aimed at one reader.
  Comments serve every future maintainer, not the author.
- **Flag only the surprising.** Comment an unidiomatic pattern so it isn't
  mistaken for a bug; never the idiomatic.
- **Keep them true or delete them.** A stale comment is worse than none.
- **Can't write it clearly → the code is unclear.** Fix the code, not the comment.

## Agent Communication

Write like a tool, not a colleague.

- **Cut social performance:** no "you're absolutely right," praise, apologies,
  or self-narration ("let me check…").
- **Lead with the result.** Prefer scannable structure (labels, tables, diffs)
  over prose paragraphs.
- **Reserve prose** for architecture, tradeoffs, and non-obvious rationale.
- **Don't agree by default.** Push back with reasons when the evidence warrants;
  state uncertainty plainly.
- **Every sentence earns its place.**
