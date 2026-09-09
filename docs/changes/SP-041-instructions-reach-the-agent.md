# SP-041 — The instruction files reach the agent

Status: implemented.
Source: a user question on 2026-09-09. Section 8.1 and Section 8.2.

## Problem

Section 8.1 says `init` writes the ShipProof instructions "into the format that
each harness reads". It does not, and the content it writes does not answer the
question a user asked.

Two defects, and either one alone makes the layer useless.

### The delivery does not work

`init` writes a loose `plan-proof.md` into `.claude/skills/`. Claude Code loads
a skill from `.claude/skills/<name>/SKILL.md` with YAML frontmatter that names
it and says when to use it. A loose Markdown file in that directory is not
loaded, so the agent never sees it.

The same holds for `.opencode/skills/` and `.agents/skills/`. ShipProof picked
a directory that looks like a skill directory and wrote something that is not a
skill.

Row T1 asked that `init` install three files for five targets. It does, and a
passing proof of delivery is not a proof of arrival.

### The content does not answer the question

`plan-proof.md` never names `verification.json`. It explains what a proof is
and what a grade means, and it never tells an agent which file to write, what
shape that file takes, or what a filled entry looks like.

An agent that reads it learns the philosophy and still cannot act.

## Requirements

R1. `init` writes each instruction as `<name>/SKILL.md` under every harness
skills directory, with frontmatter that names it and states when to use it.

R2. The frontmatter carries a `description` that an agent matches against a
task, so the file loads when it is relevant and not otherwise.

R3. `plan-proof.md` names `.shipproof/changes/<id>/verification.json`, shows
its shape, and holds one worked example of a filled entry.

R4. `plan-proof.md` shows how to mark a proof human, with the fields that
requires.

R5. `capture-intent.md` and `read-evidence.md` name the files they act on.

R6. `init` removes a loose instruction file that an earlier version wrote. A
file that no harness reads is worse than no file, because it looks installed.

R7. The five targets keep working: Claude Code, Cursor, Codex, OpenCode, and a
plain `AGENTS.md`.

## Acceptance

- A test asserts that every installed instruction is a `SKILL.md` inside its
  own directory.
- A test asserts that each one carries a `name` and a `description` in
  frontmatter.
- A test asserts that `plan-proof` names `verification.json` and holds a
  worked example that parses as the plan schema.
- A test asserts that `init` removes a loose instruction file from an earlier
  version.
- `just verify` exits 0.

## Non-goals

- A new command. `init` already installs, and Section 4 fixes the surface.
- Writing the plan for the user. ShipProof tells an agent what the file is; it
  does not guess a proof command.
