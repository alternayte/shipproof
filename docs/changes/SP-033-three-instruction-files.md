# SP-033 — Three instruction files replace the skill catalog

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 8.2, Section 14.5.
Definition of done: row T1.

## Problem

Row T1 says that `init` installs instructions for five harness targets, and
that the harness install test is "reduced to three instruction files".

`init` installs six skill packages: `prepare-change`, `plan-verification`,
`implement-change`, `produce-evidence`, `review-change`, and
`prepare-human-review`. Section 8.2 names three files and no skill catalog.

The six names also live in the code. `phase.Result.NextSkill` names one of
them for every phase, so `shipproof status` tells a reader to run a skill that
this change removes.

## Scope

Replace the six skill packages with the three instruction files of Section
8.2. Keep the five targets and the install machinery. Point the phase at the
instruction file that applies.

## Requirements

R1. `skills/` holds `capture-intent.md`, `plan-proof.md`, and
`read-evidence.md`, and nothing else.

R2. Every one of the three files states the rule that Section 8.2 calls the
one that matters most: an agent must never write a result that a tool did not
produce.

R3. `init` installs the three files for all five targets: Claude Code, Cursor,
Codex, OpenCode, and a plain `AGENTS.md`.

R4. `init` removes the six retired skill directories from a repository that
holds them.

R5. The three files name only the five commands and the two support commands
of Section 4.

R6. `phase.Result` names an instruction file, never a removed skill. The field
is `next_instruction`.

R7. No user-facing document names a removed skill.

## Acceptance

- A test asserts that the install writes exactly three files per target.
- A test asserts that each installed file holds the never-write rule.
- A test asserts that `init` removes a retired skill directory.
- A test asserts that no instruction file names a command outside Section 4.
- A test asserts that `phase` names one of the three files for every phase.
- `just verify` exits 0.

## Non-goals

- The hook contract of Section 8.1. Row T3 covers it.
- Any change to the runner interface. Row T2 is met and stays met.
