# Phase 7 decomposition — Reports

**Source:** SDD §19, §24 (Phase 7), shaping session `phase-7-reports`
**State:** READY_WITH_ASSUMPTIONS (A-001, A-002 | U-001)

## Changes

### SP-009 — Per-change reports

Establish the report package and CLI routing. Implement the HTML change
report and the Markdown PR evidence summary for a single change.

**Requirements:**

- SP-009-R1 — Render intent section (snapshot hash, requirement count)
- SP-009-R2 — Render readiness state (from change record)
- SP-009-R3 — Render assumptions and known risks
- SP-009-R4 — Render requirement coverage table
- SP-009-R5 — Render verification check results with status indicators
- SP-009-R6 — Render changed surface (commits, files, additions, deletions)
- SP-009-R7 — Render agent-run metadata (model, tokens, cost, duration)
- SP-009-R8 — Render automated findings (observed-checks section)
- SP-009-R9 — Render human attention points (from review packet)
- SP-009-R10 — Render human-review surface (unknown/skip checks)
- SP-009-R11 — Render Markdown PR summary: what changed
- SP-009-R12 — Render Markdown PR summary: deterministic evidence
- SP-009-R13 — Render Markdown PR summary: what remains uncertain
- SP-009-R14 — Render Markdown PR summary: what to inspect
- SP-009-R15 — Render Markdown PR summary: why each inspection matters
- SP-009-R16 — Separate observed/derived/inferred/human provenance in both formats
- SP-009-R17 — Wire `shipproof report change <id>` and `shipproof report pr-summary <id>` CLI commands
- SP-009-R18 — Support `--output` flag for file writing (default stdout)
- SP-009-R19 — Validate that the evidence pack exists before rendering

**Dependencies:** none (reads existing EvidencePack and ReviewPacket from disk)

---

### SP-010 — Project aggregate report

Add multi-change aggregation. Read all evidence packs in `changes/*/` and
derive project-level metrics. Render as an HTML project report.

**Requirements:**

- SP-010-R1 — Scan `.shipproof/changes/*/evidence-pack.json` and load all packs
- SP-010-R2 — Derive verification pass rate (count pass / total checks)
- SP-010-R3 — Derive first-pass success rate from `verification:run` checks
- SP-010-R4 — Derive agent usage summary (models, tokens, tool calls aggregated)
- SP-010-R5 — Derive cost summary where agent cost data exists
- SP-010-R6 — Derive requirement coverage across all changes
- SP-010-R7 — Render unavailable metrics as "no data available" (cycle time, review wait, rework, human review effort, readiness blockers)
- SP-010-R8 — Render provenance labels for each derived metric
- SP-010-R9 — Wire `shipproof report project <name>` CLI command
- SP-010-R10 — Support `--output` flag for file writing (default stdout)

**Dependencies:** SP-009 (uses report package structure, CLI routing, embed pattern, provenance rendering)

---

## Requirement coverage

| SDD requirement | SP-009 | SP-010 |
|---|---|---|
| Intent | R1 | — |
| Readiness state | R2 | — |
| Assumptions / risks | R3 | — |
| Requirement coverage | R4 | R6 |
| Verification | R5 | R2 |
| Changed surface | R6 | — |
| Agent-run metadata | R7 | R4 |
| Automated findings | R8 | — |
| Human attention points | R9 | — |
| Human-review surface | R10 | — |
| What changed (MD) | R11 | — |
| Deterministic evidence (MD) | R12 | — |
| What is uncertain (MD) | R13 | — |
| What to inspect (MD) | R14 | — |
| Why each inspection (MD) | R15 | — |
| Verification pass rate (agg) | — | R2 |
| First-pass success (agg) | — | R3 |
| Agent usage (agg) | — | R4 |
| Cost (agg) | — | R5 |
| No-data markers (agg) | — | R7 |
| Provenance separation | R16 | R8 |
| Assessment rule | R16 | R8 |

## Uncovered requirements

None. All SDD §19 items are covered.

## Dependency graph

```
SP-009 (per-change reports)
  └── SP-010 (project aggregate report)
```

SP-009 and SP-010 can be implemented sequentially. They cannot be parallel
because SP-010 reuses the report package, CLI routing, and embed pattern
established by SP-009.
