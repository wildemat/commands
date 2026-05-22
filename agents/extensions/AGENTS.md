_created_at: 2026-02-18_
_edited: 2026-05-22_

# Extensions

## Beads (task management)
_created_at: 2026-02-18_

"Beads" library for task management - Use 'bd' for task management. If no `.beads` directory is present in the project, run `bd init` to create one. - Keep the beads database current and repair any corruption issues as they arise, beads should be the source of truth for context and task management. If the beads mcp server presents issues or bugs, fallback to the `bd` cli commands.

## context7
_created_at: 2026-02-18_

Use context7 mcp server to stay aware of latest library versions and documentation

## Skill discovery prompt
_created_at: 2026-05-22_

At the **very start of your first response in a new session**, print exactly one line — before any other greeting, status, or work — in this format:

```
/skill-try <learned>/<installed>
```

Where:

- `<installed>` = sum of:
  - subdirectories under `~/.claude/skills/` that contain a `SKILL.md` file (local skills), and
  - for each plugin in `~/.claude/plugins/installed_plugins.json`: subdirectories under that plugin's most-recent `installPath/skills/` containing a `SKILL.md`. Skip plugins whose `installPath/skills/` doesn't exist.
- `<learned>` = number of entries in the JSON array at `~/.claude/skill-try-learned.json` (key: `learned`). If the file does not exist or is malformed, use `0` — do not create the file.

This line is a progress nudge: it reminds the user that the `/skill-try` skill is available and shows how many of their installed skills they've toured. Do not explain it, expand it, or annotate it. One line, then continue with the actual response.

If neither `~/.claude/skills/` nor `~/.claude/plugins/installed_plugins.json` exists, skip the line entirely.

The `/skill-try` skill itself maintains `~/.claude/skill-try-learned.json` — this directive only reads it.
