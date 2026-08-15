# SP-016 — Top-level verify command

## Problem

The SDD §23 CLI surface lists `shipproof verify` as the repository
verification entry point. The CLI only offers `verification run`, which
requires a change record. A user cannot run the configured verification
command without creating a change first.

## Desired outcome

`shipproof verify` runs the repository-owned verification command. With a
change id it behaves like `verification run`. Without one it stores logs
under `.shipproof/runs/adhoc/` and returns the command exit code.

## Scope

A small refactor of `internal/verify` to share the run logic, plus a new
`verify` CLI command.

## Requirements

### SP-016-R1 — Shared run logic

Extract the command execution from `verify.Run` into an internal helper.
`verify.Run` keeps its behavior and output format exactly. No existing
run result changes.

### SP-016-R2 — Adhoc run

`verify.RunAdhoc(root, command)` runs the configured command, writes
stdout and stderr logs under `.shipproof/runs/adhoc/`, and writes no
run.json because no change exists to associate the result with.

### SP-016-R3 — CLI command

`shipproof verify [change-id]` delegates to `verification run` when an id
is present. Without an id it runs adhoc and returns the command exit
code. Usage errors return 2. Operational errors return 1.

### SP-016-R4 — Tests

Unit tests for `RunAdhoc` including a failing command and a missing
command. CLI tests for delegation, adhoc success, exit code passthrough,
usage errors, and missing config.

## Acceptance criteria

- `just verify` passes
- `shipproof verify` returns the verification command's exit code
- `shipproof verify SP-016` writes a run.json identical to
  `verification run`
- Adhoc runs never create a run.json

## Non-goals

- A separate verification command list beyond the configured one
- CI integration modes

## Dependencies

SP-002 (verification runner).
