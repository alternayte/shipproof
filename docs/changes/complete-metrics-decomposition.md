# Complete metrics decomposition

**Source:** PRD `docs/prd/complete-metrics.md`, SDD
`docs/sdd/complete-metrics-sdd.md`, shaping sessions `complete-metrics` and
`complete-metrics-sdd`
**State:** READY_WITH_ASSUMPTIONS (SDD assumptions A1, A2; risks R1-R4)

## Changes

### SP-011 — Cycle time and rework metrics

Derive cycle time and rework from existing pack data. No schema change.
Incorporates the uncommitted `cycleTimeForPack` WIP in `project.go`.

**Requirements:** SP-011-R1..R8

### SP-012 — Readiness blocker snapshot

Optional `ReadinessEvidence` schema field. Assembler embeds the blocker
count from the shaping session named by `ShapingRef`. Report renders per
change and totals. Incorporates the uncommitted `ShapingRef` WIP.

**Requirements:** SP-012-R1..R6

### SP-013 — GitHub review client and command

`internal/github` GraphQL client, git origin URL resolution, and the
`shipproof evidence review` command writing `review.json`.

**Requirements:** SP-013-R1..R5

### SP-014 — Review metrics end-to-end

Optional `ReviewEvidence` schema field, assembler merge, review wait and
human review effort derivation, and removal of the unavailable section.

**Requirements:** SP-014-R1..R6

## Requirement coverage

| PRD metric | Change |
|---|---|
| Cycle time | SP-011 (R1, R2) |
| Rework rate | SP-011 (R3, R4) |
| Readiness blockers | SP-012 (R3, R4) |
| Review wait | SP-014 (R3) |
| Human review effort | SP-014 (R4) |

| PRD acceptance criterion | Covered by |
|---|---|
| All ten metrics displayed | SP-011, SP-012, SP-014 |
| Five metrics show real derived data | SP-011, SP-012, SP-014 |
| No unavailable section remains | SP-014-R5 |
| Provenance badge per new metric | SP-011-R5, SP-012-R5, SP-014-R5 |
| Specific gap notices | SP-011-R1, SP-014-R3, SP-014-R4 |
| Existing metrics unchanged | Existing test suite in every change |

## Uncovered requirements

None. All PRD metrics and acceptance criteria map to a change.

## Dependency graph

```
SP-011 (cycle time + rework)
SP-012 (readiness blockers)
SP-013 (GitHub review client)
  └── SP-014 (review metrics end-to-end)
```

SP-011, SP-012, and SP-013 are independent. SP-014 depends on SP-013 for
the `review.json` producer and the `ReviewEvidence` shape.

## Suggested order

SP-011, SP-012, SP-013, SP-014. SP-011 and SP-012 first because they
consume the uncommitted WIP and need no network access.
