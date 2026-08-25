# SP-021 — Per-proof execution and the coverage matrix

## Problem

`verification run` runs one repository gate command and records one exit code.
An agent then writes a per-requirement status and labels it `observed`. Nothing
ran a proof for that requirement, so the label is not honest. The v1 spine
Section 1 names this the serious consequence.

## Desired outcome

Every proof in the verification plan runs on its own. One result per proof
lands in `.shipproof/runs/<id>/proofs.json`. A coverage matrix derives the
state of each requirement from those results. No state reads `inferred`.

## Scope

The plan proof gains a human form. The runner gains an attribution pass. A new
artifact holds the per-proof results. A new command renders the matrix.

## Requirements

### SP-021-R1 — Human proofs in the plan

A plan proof carries either a non-empty `command` or `human: true` with a
non-empty `rationale`. `verification check` rejects a proof that carries
neither. A human proof can carry `accepted_at` with an RFC 3339 timestamp. A
proof that carries a command cannot carry `accepted_at`.

### SP-021-R2 — Shared run currency

One function judges whether a recorded run still describes the working tree.
`internal/phase` and the coverage matrix both use it. A run with no recorded
revision cannot be judged and is never stale. The tree check excludes
`.shipproof/`.

### SP-021-R3 — The proofs artifact

`.shipproof/runs/<id>/proofs.json` holds `schema_version`, `change_id`,
`head_rev`, `tree_clean`, `timestamp`, and one `results` entry per proof in the
plan. Each entry holds `requirement_id`, `proof_index`, `command`, `exit_code`,
`duration_ms`, and `status`. A JSON Schema validates the artifact.

### SP-021-R4 — Per-proof execution

`shipproof verification run <id>` runs the repository gate and then runs each
automated proof on its own. It records one result per proof. A human proof
records the status `human` and runs no command. `--gate-only` skips the
attribution pass. `--proofs-only` skips the gate.

### SP-021-R5 — The coverage matrix

`shipproof coverage <id> [--json]` derives one row per requirement in the
requirement sidecar. A row reads `proven`, `failed`, `accepted`,
`awaiting-human`, or `unproven`. Provenance reads `observed`, `human`, or
`unknown`. No row reads `inferred`. The matrix is derived on demand and no
command writes it.

### SP-021-R6 — Documentation

`docs/workflow.md`, `README.md`, and the `plan-verification` and
`implement-change` skills describe the human proof, the attribution pass, and
the coverage command.
