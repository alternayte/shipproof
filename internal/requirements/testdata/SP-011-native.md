# SP-011 — Cycle time and rework metrics

## Problem

Reports carry no delivery timing.

## Requirements

### SP-011-R1 — Derive per-change cycle time

Compute the elapsed time from the first commit to the merge commit.

### SP-011-R2 — Derive project average cycle time

Average the per-change values across the project.

### SP-011-R3 — Derive per-change rework

Count the commits that touch a file an earlier commit in the change touched.

### SP-011-R4 — Derive project average rework

Average the per-change rework across the project.

### SP-011-R5 — Render metric cards

Add a card for each metric to the HTML report.

### SP-011-R6 — Extend summary table

Add a column for cycle time and a column for rework.

### SP-011-R7 — Remove unavailable placeholders

Omit a metric that the repository cannot supply.

### SP-011-R8 — Tests

Cover each derivation with a table-driven test.

## Acceptance criteria

The report renders both metrics.

## Non-goals

No forecast.
