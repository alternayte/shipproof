# SP-011 — Cycle time and rework metrics

## Problem

The project aggregate report marks cycle time and rework rate as unavailable.
The evidence packs already contain commit timestamps and commit counts. Both
metrics can be derived today without any schema change.

## Desired outcome

`shipproof report project <name>` derives cycle time per change (oldest
commit timestamp to evidence pack generation) and rework per change (commit
count). The report shows project averages with derived provenance badges.

## Scope

Extend `internal/report/project.go` and the project report template only.
No schema change. Incorporate the uncommitted `cycleTimeForPack` and
`formatCycleDuration` functions that already exist in `project.go`.

## Requirements

### SP-011-R1 — Derive per-change cycle time

Compute duration from the oldest commit timestamp in
`ImplementationEvidence.commits` to `provenance.generated_at`. Show a gap
notice when commit data is missing or timestamps do not parse.

### SP-011-R2 — Derive project average cycle time

Average cycle time across changes with valid values. Ignore changes with
gap notices. Label as derived.

### SP-011-R3 — Derive per-change rework

Count commits in `ImplementationEvidence.commits`. Show the integer count
per change.

### SP-011-R4 — Derive project average rework

Average commits per change across all changes. Label as derived.

### SP-011-R5 — Render metric cards

Add cycle time and rework cards to the project report template with derived
provenance badges.

### SP-011-R6 — Extend summary table

Add per-change columns for cycle time and commit count, including gap
notices in rows.

### SP-011-R7 — Remove unavailable placeholders

Remove "Cycle time" and "Rework rate" from the hard-coded unavailable
metrics list.

### SP-011-R8 — Tests

Unit tests for both derivation functions with fixture packs, following
`project_test.go`. Template content assertions for the new cards and
columns.

## Acceptance criteria

- `just verify` passes
- Cycle time and rework values appear in the project report
- A pack with no commits shows a cycle time gap notice, not a crash
- Existing metrics R2-R6 still render unchanged
- "Cycle time" and "Rework rate" no longer appear in the unavailable section

## Non-goals

- Schema extension
- Storing derived durations in the evidence pack
- Trend analysis

## Dependencies

None. Reads existing evidence packs only.
