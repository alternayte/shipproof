# SP-020 — Skill eval runs and benchmark skill

## Problem

SDD §11 and §21 require every released skill to have an eval set, run
with-skill versus without-skill comparisons, and detect regressions.
Exit criteria 13 and 25 depend on this. The repository has eval case
fixtures but no recorded runs, no `benchmark-run` skill, and no
`benchmarks/` directory.

## Desired outcome

Eval runs can be recorded deterministically with validation and compared
for regressions. A `benchmark-run` skill documents the implementation
benchmark procedure. Recorded with-skill and without-skill runs exist
for the spec-skill cases.

## Scope

A versioned eval run record in the skills package, `skill eval record`
and `skill eval results` CLI commands, a `benchmark-run` skill, and a
`benchmarks/` directory layout.

## Requirements

### SP-020-R1 — Eval run schema

Define a versioned eval run record with condition (`without`,
`previous`, `candidate`), task success, blocker recall, false blocker
rate, questions asked, human intervention, and optional elapsed time,
tokens, observed cost, and human rating. Unknown fields stay absent.

### SP-020-R2 — Recording

`shipproof skill eval record <case-id> --condition <c> --file <json>`
validates the run against the built-in eval manifest and stores it
under `benchmarks/skill-evals/<case-id>/<condition>/run-<n>.json`.

### SP-020-R3 — Results and regression

`shipproof skill eval results [case-id] [--regression]` lists recorded
runs. A regression is reported when a candidate run fails a task the
baseline passed, recalls fewer blockers, or raises the false blocker
rate. Regression detection returns exit code 1.

### SP-020-R4 — Benchmark skill

Add the `benchmark-run` portable skill for implementation benchmark
tasks: fixed starting commits, hidden evaluation material the agent
must not open, metric recording, and keeping failed runs.

### SP-020-R5 — Recorded runs

Record with-skill and without-skill runs for the spec-skill cases
`prd-ready-stop`, `prd-hidden-solution`, `sdd-cargo-cult-broker`, and
`sdd-real-concurrency-blocker`. Judge each run against the case
expected and penalized behaviors. No result is fabricated.

### SP-020-R6 — Tests

Schema validation tests, recording and loading tests, regression
detection tests, and CLI record and results tests.

## Acceptance criteria

- `just verify` passes
- Eight recorded runs exist across the four spec-skill cases
- `skill eval results --regression` reports no regression
- `skill check` validates the catalog including `benchmark-run`

## Non-goals

- Automatic agent invocation to run evals
- Hidden-test scoring for implementation benchmarks

## Dependencies

SP-008 (agent telemetry metrics list), Phase 3 skill catalog.
