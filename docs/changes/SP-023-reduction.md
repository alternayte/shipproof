# SP-023 — Reduction

## Problem

The SDD cuts six packages, one eval subsystem, and eight portable skills. That
code serves the old strategy. It duplicates OpenSpec and Kiro, it adds two
network adapters that produce no evidence, and it carries a personal writing
standard as a product feature. The code still ships in the binary and the
help output still advertises it.

## Desired outcome

Every cut package is gone from the tree. Every cut command exits with code 2
and prints one line that names the replacement or states that the feature is
gone. No dead import and no dead flag remain. The binary builds and the suite
passes.

## Scope

This change is step 1 of Section 16 of the SDD. It removes code. It adds no
feature.

Delete these packages with their tests: `internal/shaping`,
`internal/document`, `internal/language`, `internal/plan`, `internal/linear`,
`internal/github`. Delete the eval subsystem inside `internal/skills` and keep
the catalog validator.

Delete these portable skills: `shape-prd`, `shape-sdd`, `review-prd`,
`review-sdd`, `decompose-plan`, `triage-change`, `record-decision`,
`benchmark-run`. Delete the eval case file and the recorded eval runs.

Delete the shaping session schema and its test data. The evidence pack schema
does not change in this step. Section 16 step 4 reshapes the pack.

Replace these verbs with a stub: `doc`, `shape`, `plan`, `linear`,
`evidence review`, `skill eval`.

## Requirements

### SP-023-R1 — The cut packages are gone

The six cut package directories do not exist. The eval subsystem inside
`internal/skills` does not exist.

### SP-023-R2 — Every cut verb exits 2 with one line

`shipproof doc`, `shipproof shape`, `shipproof plan`, `shipproof linear`,
`shipproof evidence review`, and `shipproof skill eval` each exit with code 2
and print exactly one line on standard error. The line names the replacement
or states that the feature is gone.

### SP-023-R3 — No dead import and no dead flag

The `change start` command does not accept `--shaping`. The change record
does not hold a shaping reference. The evidence pack assembler does not read a
shaping session and does not read a review file.

### SP-023-R4 — The help output drops every cut verb

The usage text names no cut verb.

### SP-023-R5 — The documentation drops every cut reference

`docs/workflow.md` and `benchmarks/README.md` name no cut skill and no cut
command.

## Acceptance

- `test ! -d internal/shaping -a ! -d internal/document -a ! -d internal/language -a ! -d internal/plan -a ! -d internal/linear -a ! -d internal/github` exits 0.
- A CLI test asserts the exit code and the one-line message for each cut verb.
- A CLI test asserts that the usage text names no cut verb.
- `just verify` exits 0.

## Non-goals

- The five-command surface of Section 4. Section 16 step 2 covers it.
- The verdict block and the three grades. Section 16 step 3 covers it.
- The pack reshape and the schema version change. Section 16 step 4 covers it.
- The report rebuild. Section 16 step 8 covers it.
