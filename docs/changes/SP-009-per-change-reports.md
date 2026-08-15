# SP-009 — Per-change reports

## Problem

Evidence packs and review packets sit on disk as JSON. The `shipproof report`
command does not exist. A human reviewer must read raw JSON to understand
what changed, what is proven, and what needs attention.

## Desired outcome

`shipproof report change <id>` renders a single-scrolling-page HTML change
report from the evidence pack and review packet. `shipproof report
pr-summary <id>` renders a sectioned Markdown PR evidence summary that answers
the five SDD review questions. Both formats display provenance tags on every
metric.

## Scope

Create `internal/report/` with embedded HTML templates and generation logic.
Wire `internal/cli/report.go` with `shipproof report change <id>` and
`shipproof report pr-summary <id>` subcommands. Support a `--output` flag
for file writing; default to stdout. Do not implement the project aggregate
report in this change.

## Requirements

### SP-009-R1 — Render intent section

Read the evidence pack. Display the snapshot hash and requirement count.
Each value carries a provenance tag.

### SP-009-R2 — Render readiness state

Read the change record. Display the readiness field (ready, blocked, etc.)
with a provenance tag.

### SP-009-R3 — Render assumptions and known risks

Read the shaping session file when it exists. Display assumptions and risks
with provenance tags.

### SP-009-R4 — Render requirement coverage table

Display each requirement ID and its verification refs. Show coverage status.
Each value carries a provenance tag.

### SP-009-R5 — Render verification check results

Display all checks from the evidence pack with status (pass/fail/skip/unknown)
and source. Color-code by status. Each check carries a provenance tag.

### SP-009-R6 — Render changed surface

Display commits, changed files, additions, deletions, and diff stat from the
implementation evidence. Each metric carries a provenance tag.

### SP-009-R7 — Render agent-run metadata

Display provider, model, session ID, cost, token usage, tool call count,
duration, and exit status from the agent run metadata. Mark missing data
explicitly. Each metric carries a provenance tag.

### SP-009-R8 — Render automated findings

Display observed-pass provenance checks as the automated-findings section.
These are the deterministically proven items.

### SP-009-R9 — Render human attention points

Display human-attention checks from the review packet with their reason and
relevant requirements. Each item carries a provenance tag.

### SP-009-R10 — Render human-review surface

Display unknown and skipped checks as the review-surface section. Show what
is uncertain about each.

### SP-009-R11 — Render Markdown PR summary: what changed

Read the evidence pack and review packet. Write a `## What changed` section
with a summary of changed files, commit count, additions, and deletions.

### SP-009-R12 — Render Markdown PR summary: deterministic evidence

Write a `## Deterministic evidence` section listing already-proven checks
from the review packet.

### SP-009-R13 — Render Markdown PR summary: what remains uncertain

Write a `## What remains uncertain` section listing unknown and skipped
checks with their uncertainty descriptions.

### SP-009-R14 — Render Markdown PR summary: what to inspect

Write a `## What to inspect` section listing human-attention checks.

### SP-009-R15 — Render Markdown PR summary: why each inspection matters

Include the reason for each human-attention check. Reference relevant
requirements.

### SP-009-R16 — Separate provenance in both formats

In HTML, render provenance as colored badges: observed (green), derived
(blue), inferred (yellow), human (purple). In Markdown, render provenance
as inline tags: `[observed]`, `[derived]`, `[inferred]`, `[human]`.

### SP-009-R17 — Wire CLI commands

Add `shipproof report change <id>` and `shipproof report pr-summary <id>`
to the CLI router in `internal/cli/app.go`. Add usage text to `printUsage`.

### SP-009-R18 — Support --output flag

Both subcommands accept an optional `--output <path>` flag. When given,
write the report to the file path instead of stdout. Create parent
directories as needed.

### SP-009-R19 — Validate evidence pack existence

Before rendering, verify that `.shipproof/changes/<id>/evidence-pack.json`
exists. Return an error with exit code 1 when it does not.

## Acceptance criteria

- `shipproof report change SP-005` writes valid HTML to stdout
- `shipproof report change SP-005 --output report.html` writes to file
- `shipproof report pr-summary SP-005` writes valid Markdown to stdout
- Every metric in the HTML report carries a provenance badge
- Every metric in the Markdown summary carries an inline provenance tag
- Missing evidence pack produces an error with exit code 1
- No project aggregate functionality is included

## Non-goals

- Project aggregate report (SP-010)
- Tabbed or collapsible HTML sections
- User-customizable templates
- Schema extension for new metric types
- Report output other than HTML and Markdown

## Dependencies

None. Reads existing EvidencePack and ReviewPacket from disk.
