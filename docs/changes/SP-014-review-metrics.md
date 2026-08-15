# SP-014 — Review metrics end-to-end

## Problem

Review wait and human review effort remain unavailable in the project
aggregate report. SP-013 provides the `review.json` data file. The evidence
pack schema, assembler, and report must connect that data to the report.

## Desired outcome

Evidence pack assembly merges `review.json` into the pack. The project
report derives review wait and human review effort per change, shows
project aggregates, and removes the unavailable metrics section entirely.

## Scope

Schema `ReviewEvidence` field. Assembler merge with a `github:review`
check. Report derivation and template rendering.

## Requirements

### SP-014-R1 — Schema field

Add `Review *ReviewEvidence` to `EvidencePack` with
`json:"review,omitempty"` per SDD §3.2. Validate RFC 3339 timestamps when
present. Keep schema version 0.1.

### SP-014-R2 — Assembler merge

In `Assemble`, read `.shipproof/changes/<change-id>/review.json`. Skip when
missing. Set `pack.Review` when present. Append a `github:review` check
with status `pass`, source `github`, observed provenance. Return an error
for a malformed file.

### SP-014-R3 — Review wait derivation

Per change: start is `pack.AgentRun.EndedAt` when present, else the latest
commit timestamp. End is `pack.Review.FirstReviewAt`. Gap notice when
review data is nil, `FirstReviewAt` is empty, timestamps do not parse, or
end is before start. Project average across valid values. Label as derived.

### SP-014-R4 — Human review effort derivation

Per change: review count, comment count, distinct reviewers from
`pack.Review`. Project totals and distinct reviewer union across changes.
Label as derived.

### SP-014-R5 — Template

Add review wait and human review effort cards and table columns. Remove
the unavailable metrics section and its styles entirely.

### SP-014-R6 — Tests

Schema tests for the new field. Assembler tests for merge, skip, and
malformed paths. Report tests for both metrics including all gap notice
cases. Template assertions.

## Acceptance criteria

- `just verify` passes
- Review wait and human review effort render with real data when
  `review.json` was collected
- Packs without review data show specific gap notices per change
- The "Unavailable Metrics" section no longer exists in the report
- All ten SDD §19 metrics render with provenance badges

## Non-goals

- Review data refresh at report time
- Per-thread or per-line comment display

## Dependencies

SP-013 (produces `review.json` in the `ReviewEvidence` shape).
