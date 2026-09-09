# ShipProof agent instructions

Section 8.2 of the design document names three instruction files. They replace
the old skill catalog. Three files are small enough that an agent reads all of
them. A catalog was not.

| Package | What it covers |
|---|---|
| `capture-intent/SKILL.md` | How to record the intent document and its requirements. |
| `plan-proof/SKILL.md` | How to map each requirement to a runnable proof in `verification.json`, with a worked example. |
| `read-evidence/SKILL.md` | How to read a pack, and how to report the verdict without inflating it. |

Each one is a skill package with frontmatter, because a harness loads
`<name>/SKILL.md` and ignores a loose Markdown file beside it. A file that no
harness reads is worse than no file, because it looks installed.

One rule sits in all three files.

> An agent must never write a result that a tool did not produce.

`shipproof init` installs these files for five harness targets: Claude Code,
Cursor, Codex, OpenCode, and a plain `AGENTS.md`. The files hold no
harness-specific content.
