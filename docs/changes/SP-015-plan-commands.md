# SP-015 — Plan commands

## Problem

The SDD §23 CLI surface lists `plan create`, `plan review`, and
`plan sync --linear`. The CLI has no `plan` command. A design document
cannot become an approved plan record through the CLI. Exit criterion 14
depends on this path.

## Desired outcome

A user can snapshot a design document as a versioned plan record, validate
all plan records deterministically, and sync a decomposed issue list to
Linear after human approval.

## Scope

A new `internal/plan` package, a `plan` CLI command group, and a reuse of
the existing `linear sync` path.

## Requirements

### SP-015-R1 — Plan record schema

Define a versioned plan record under `.shipproof/plans/<plan-id>/plan.json`
with schema version 0.1: plan id, source path, snapshot path, SHA-256,
creation time, and optional change ids. Validate the record on load.

### SP-015-R2 — `plan create`

`shipproof plan create <file>` derives a plan id from the file name,
snapshots the source content, and writes the record. Duplicate plans and
names that cannot form a valid id produce clear errors.

### SP-015-R3 — `plan review`

`shipproof plan review` validates every plan record: schema, snapshot hash
integrity, and source staleness. An empty plans directory is a valid
result. Invalid records exit with code 1.

### SP-015-R4 — `plan sync --linear`

`shipproof plan sync --linear [plan-file]` delegates to the existing
Linear sync flow. Without a file it uses the single `issues.json` under
`.shipproof/plans/`. Zero or multiple candidates produce clear errors.
Human approval remains required inside the sync flow.

### SP-015-R5 — Tests

Unit tests for record lifecycle, staleness, listing, and id derivation.
CLI tests for create, review (valid, stale, empty), and sync error paths.

## Acceptance criteria

- `just verify` passes
- `plan create` produces a record with a matching snapshot hash
- `plan review` reports valid, stale, and invalid plans distinctly
- `plan sync` fails with a clear error when no `issues.json` exists

## Non-goals

- Automatic decomposition of documents into issues
- Linear relationship management beyond the existing sync flow

## Dependencies

SP-007 (Linear adapter and sync flow).
