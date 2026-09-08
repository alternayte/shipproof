# SP-024 — The command surface

## Problem

Section 4 of the SDD states that the whole product is five commands plus two
support commands. The binary publishes seventeen verbs. A user must chain four
commands to reach a result. The help output reads as a toolbox, not as a
product.

## Desired outcome

The binary publishes five commands, two support commands, and the agent
execution command. Every folded verb exits with code 2 and prints one line
that names its replacement. A user reaches a full result with `start` and
then `pack`.

## Scope

This change is step 2 of Section 16 of the SDD. It re-wires the surface over
the machinery that already exists. It changes no proof logic and no pack
format.

The new surface:

```
shipproof init [directory]
shipproof start <change-id> --intent <file>
shipproof prove [change-id]
shipproof pack [change-id]
shipproof status [change-id]
shipproof runner <list|doctor>
shipproof config <get|set> <key> [value]
shipproof run <change-id>
```

The folds:

| Old verb | New home |
|---|---|
| `harness install` | `init` |
| `change start` | `start` |
| `change status`, `change check`, `next`, `coverage` | `status` |
| `verify`, `verification run`, `verification check` | `prove` |
| `verification init` | `status`, which names the missing plan |
| `evidence pack`, `telemetry collect`, `report change` | `pack` |

The deletions: `report project`, `report pr-summary`, `review prepare`, and
`skill check`, with `internal/review` and `internal/skills`. Section 16 step 6
rebuilds the pull-request comment inside the action. Section 15 decision D5
returns the portfolio report later as a consulting artifact.

`start` adopts the requirement set with the native adopter that already
exists. SP-025 replaces that adopter with the single documented pattern of
Section 15 decision D3.

## Requirements

### SP-024-R1 — The public surface is eight verbs

The help output names `init`, `start`, `prove`, `pack`, `status`, `runner`,
`config`, and `run`. It names no other verb.

### SP-024-R2 — Every folded verb exits 2 with one line

`verification`, `verify`, `harness`, `change`, `next`, `coverage`, `skill`,
`evidence`, `review`, `telemetry`, and `report` each exit with code 2 and
print one line that names the replacement.

### SP-024-R3 — `init` installs the harness instructions

`init` prepares the repository and installs the ShipProof instructions for
the harness that the directory holds.

### SP-024-R4 — `start` snapshots the intent and adopts the requirements

`start <change-id> --intent <file>` records the intent snapshot with its hash.
It writes the requirement set when the document holds one. It states the count
it adopted.

### SP-024-R5 — `prove` runs the gate and each proof

`prove` validates the verification plan, runs the repository gate, and runs
each proof on its own. It records one result per proof.

### SP-024-R6 — `pack` assembles the pack and renders the report

`pack` collects the agent telemetry when a record exists, assembles the
evidence pack, and writes the HTML change report.

### SP-024-R7 — `status` reports the phase, the blocker, and the coverage

`status` prints the current phase, the blocker, the next command, and the
requirement coverage.

### SP-024-R8 — Two commands reach a full result

`start` then `pack` produces an evidence pack from a clean repository.

## Acceptance

- A CLI test asserts that the help output names the eight verbs and no other.
- A CLI test asserts the exit code and the one-line message for each folded verb.
- A CLI test runs `start` then `pack` and asserts that the pack exists.
- `just verify` exits 0.

## Non-goals

- The verdict block and the three grades. Section 16 step 3 covers them.
- The pack reshape and the schema version change. Section 16 step 4 covers them.
- The single documented requirement pattern of decision D3. SP-025 covers it.
- The portfolio report. Decision D5 returns it outside the definition of done.
