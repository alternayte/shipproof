# SP-010 — Project aggregate report

## Problem

Per-change reports (SP-009) show evidence for a single change. There is no
way to see project-level metrics such as verification pass rate, agent usage,
or cost across all changes in a repository.

## Desired outcome

`shipproof report project <name>` scans all `changes/*/evidence-pack.json`
files, derives aggregate metrics from the available data, and renders an HTML
project report. Metrics that cannot be derived from the current schema display
"no data available" with appropriate provenance.

## Scope

Add project aggregate generation to the existing `internal/report/` package.
Add `shipproof report project <name>` to the CLI. Scan all evidence packs
in `.shipproof/changes/*/`, derive aggregate metrics, and render an HTML
report with embedded provenance tags. Do not extend the evidence pack schema.

## Requirements

### SP-010-R1 — Scan and load all evidence packs

Read every `evidence-pack.json` file under `.shipproof/changes/*/`. Skip
directories where the file does not exist. Return an error when no evidence
packs are found.

### SP-010-R2 — Derive verification pass rate

Count pass / total checks across all evidence packs. Display the pass rate
as a percentage. Label each metric with `derived` provenance.

### SP-010-R3 — Derive first-pass success

Count changes where `verification:run` check status is `pass`. Display the
count and percentage. Label as `derived` provenance.

### SP-010-R4 — Derive agent usage summary

Aggregate provider, model, total input tokens, total output tokens, and
total tool call count across all evidence packs with agent run data. Count
changes with agent data and changes without. Label as `derived` provenance.

### SP-010-R5 — Derive cost summary

Sum agent costs across all evidence packs. Display total cost. Mark as
`derived` provenance. Note when cost data is missing for some packs.

### SP-010-R6 — Derive requirement coverage

Count total requirements and requirements with verification refs across all
packs. Display the coverage ratio. Label as `derived` provenance.

### SP-010-R7 — Mark unavailable metrics

Display "no data available" for cycle time, review wait, rework, human review
effort, and readiness blockers. Label each with appropriate provenance noting
the data gap.

### SP-010-R8 — Render provenance labels

Every derived metric and data point in the HTML project report must carry a
provenance badge.

### SP-010-R9 — Wire CLI command

Add `shipproof report project <name>` to the report CLI subcommand router and
usage text.

### SP-010-R10 — Support --output flag

Accept the same `--output <path>` flag as the per-change reports. Default to
stdout when the flag is absent.

## Acceptance criteria

- `shipproof report project my-project` writes valid HTML to stdout
- The report includes a table of all changes found
- Verification pass rate, first-pass success, agent usage, and cost are derived
- Unavailable metrics display "no data available"
- Every derived metric carries a provenance badge
- An empty repository (no evidence packs) produces a clear error
- `--output` writes to the specified file path

## Non-goals

- Schema extension for new metric collection
- Cross-repository aggregation
- User-customizable aggregate templates
- Time-series or trend analysis

## Dependencies

SP-009 (reuses the report package, embed pattern, provenance rendering, and
CLI routing structure).
