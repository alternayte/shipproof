# SP-012 — Readiness blocker snapshot

## Problem

The project aggregate report marks readiness blockers as unavailable.
Shaping sessions already record blockers in `.shipproof/shaping/<id>.json`.
The evidence pack does not carry the blocker count, so the report cannot
show it.

## Desired outcome

Evidence pack assembly reads the shaping session named by the change
record's `ShapingRef`, counts the blockers, and embeds the count in the
pack. The project report displays per-change blocker counts and a project
total.

## Scope

Add an optional `ReadinessEvidence` field to the schema (version stays
0.1). Extend the assembler. Extend the report. Incorporate the uncommitted
`ShapingRef` work in `internal/change` and the `--shaping` CLI flag.

## Requirements

### SP-012-R1 — Schema field

Add `Readiness *ReadinessEvidence` to `EvidencePack` with
`json:"readiness,omitempty"`. `ReadinessEvidence` holds `shaping_ref` and
`blocker_count`. Validate the shaping ref format when present. Keep schema
version 0.1. Existing packs stay valid.

### SP-012-R2 — Shaping ref on change records

Complete the uncommitted `ShapingRef` field on `change.Record` and the
`--shaping` flag on `shipproof change start`. Update the status output.

### SP-012-R3 — Assembly snapshot

In `Assemble`, when `record.ShapingRef` is set, load the shaping session
with `shaping.Load`, count `readiness.blockers`, and set `pack.Readiness`.
Skip silently when the ref is empty or the session file is missing.

### SP-012-R4 — Report derivation

Per change: `pack.Readiness.BlockerCount` when present, else zero. Project
total: sum across changes. Label as derived.

### SP-012-R5 — Template

Add a readiness blockers card and a per-change table column with derived
provenance. Remove "Readiness blockers" from the unavailable list.

### SP-012-R6 — Tests

Schema validation tests for the new field. Assembler tests for the
snapshot path and the skip paths. Report tests for zero and nonzero counts.

## Acceptance criteria

- `just verify` passes
- A pack assembled from a change with a shaping ref embeds the blocker
  count
- A pack without a shaping ref remains valid and the report shows zero
- "Readiness blockers" no longer appears in the unavailable section

## Non-goals

- Storing blocker IDs in the pack
- Live reading of shaping sessions at report time

## Dependencies

None.
