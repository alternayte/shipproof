# Complete project report metrics

**Status:** ready_with_assumptions
**Date:** 2026-08-15
**Shaping session:** `.shipproof/shaping/complete-metrics.json`

## Problem

The project aggregate report (`shipproof report project`) shows five of the
ten metrics listed in SDD §19. The other five display "no data available"
placeholders. A user wants to see all ten metrics with real data.

## Desired outcome

`shipproof report project <name>` derives and renders all ten aggregate
metrics from available data. No unavailable placeholders remain.

## Users

Developers and reviewers who use `shipproof report project` to assess
AI-assisted delivery quality across changes.

## Metrics to add

Five metrics are currently unavailable. Each must become available:

### Cycle time

Duration from the earliest commit in the change to evidence pack generation.

- **Start:** timestamp of the oldest commit in `ImplementationEvidence.commits`
- **End:** `provenance.generated_at` in the evidence pack
- **Display:** duration in minutes, hours, or days. The project card shows
  the mean across changes with data.
- **Provenance:** derived
- **Note:** re-assembling the evidence pack after review rewrites
  `provenance.generated_at` and extends the measured cycle time to include the
  review period. This interaction is accepted.
- **Fallback:** if no commits exist in the evidence pack (no git base revision
  was provided), display "No commit data available" with a gap notice

### Review wait

Duration the change waited for human review after implementation work ended.

- **Start:** timestamp of the latest agent run `ended_at` (the end of automated
  work). If no agent run exists, use the latest commit timestamp in
  `ImplementationEvidence.commits`. If neither exists, emit a gap notice.
- **End:** earliest timestamp across PR reviews (`/pulls/{id}/reviews`),
  PR comments (`/issues/{id}/comments`), and inline review comments
  (`/pulls/{id}/comments`) on the associated GitHub PR
- **Data source:** GitHub PR review and comment APIs, queried at evidence-pack
  generation time
- **Provenance:** derived
- **Zero rule:** if the computed duration is negative (the earliest review
  activity happened before the end of automated work), report zero. Review
  activity that started before automated work ended counts as no wait.
- **Display:** duration in minutes, hours, or days. The project card shows
  the mean across changes with data. Clamped zero values enter the mean.
- **Fallback:** if no agent run and no commits exist, display "No implementation
  timeline available" with a gap notice. If no PR is found or no review
  activity exists, display "No review data available" with a gap notice.

### Rework rate

Volume of revision activity per change.

- **Code label:** "Rework rate" (as used in `buildUnavailableMetrics()`)
- **Definition:** total commit count in `ImplementationEvidence.commits`
- **Display:** per-change integer count. A higher count implies more revision
  activity. The project card shows the sum across changes with data.
- **Provenance:** derived
- **Fallback:** if no commits exist in the evidence pack, display
  "No commit data available" with a gap notice
- **Limitation:** commit count is a coarse proxy. Squashed histories undercount
  rework. Granular commit styles overcount it. This metric captures visible
  revision volume, not actual rework effort.

### Human review effort

Amount of human attention the change received during review.

- **Definition:** total count of review comments (across PR reviews, PR
  comments, and inline review comments) and count of distinct reviewers
  across the same three endpoints. Any user who reviewed or commented on the
  associated GitHub PR counts as a reviewer.
- **Data source:** GitHub PR review and comment APIs (`/pulls/{id}/reviews`,
  `/issues/{id}/comments`, `/pulls/{id}/comments`), queried at evidence-pack
  generation time
- **Display:** per-change comment count and distinct reviewer count. The
  project card shows the sums across changes with data.
- **Provenance:** derived
- **Fallback:** if no PR is found, display "No review data available" with a
  gap notice
- **Limitation:** each GitHub list endpoint returns at most 100 items per
  page. The assembler fetches only the first page. For PRs with more than
  100 comments on a single endpoint, the reported count is capped at 100.
  This is sufficient for the vast majority of changes.

### Readiness blockers

Readiness profile from the shaping process, including blockers, decisions
required, assumptions, risks, and unknowns.

- **Definition:** counts from the shaping session associated with the change.
  Blockers and decisions required are combined into one "blocker" count (both
  prevent the session from reaching `ready` state). Assumptions, risks, and
  unknowns are reported separately.
- **Struct fields:** `Readiness.Blockers` + `Readiness.DecisionsRequired`
  (combined blocker count), `Assumptions`, `Risks`, `Unknowns`
- **Data source:** shaping session loaded via the change record's `ShapingRef`
  field (JSON key: `shaping_ref`). Resolves to
  `.shipproof/shaping/<shaping-ref>.json`.
- **Display:** per-change blocker count, assumption count, risk count, and
  unknown count. For example, "0 blockers, 3 assumptions, 1 risk, 0 unknowns".
  The project card shows the sums across changes with data.
- **Provenance:** derived
- **Fallback:** if no `shaping_ref` is set or the referenced session does not
  exist, display "No shaping data available" with a gap notice
- **Data access note:** the project report must load the change record
  (`.shipproof/changes/<id>/change.json`) in addition to the evidence pack to
  read `shaping_ref`. This is a new data access path for the report.

## Project-level display

The project report renders the five new metrics as aggregate stat cards in a
new "Delivery quality" section. The display rules:

- Cycle time and review wait show the mean across changes with data. Changes
  without data do not enter the mean.
- Rework rate, human review effort, and readiness blockers show the sum
  across changes with data.
- Each card states how many changes contributed data. If a change lacks data,
  the card lists the change ID with its gap notice.
- If no change has data, the card shows only the gap notice.
- Each card carries a derived provenance badge.

## Data sources

| Metric | Source | Collection timing |
|--------|--------|-------------------|
| Cycle time | `ImplementationEvidence.commits` timestamps + `provenance.generated_at` | Already in evidence pack |
| Review wait | GitHub PR review and comment APIs | At evidence-pack generation |
| Rework rate | `ImplementationEvidence.commits` count | Already in evidence pack |
| Human review effort | GitHub PR review and comment APIs | At evidence-pack generation |
| Readiness blockers | Change record (`shaping_ref`) + shaping session JSON | At report time |

## GitHub PR lookup strategy

GitHub data is fetched during `Assemble()` in the evidence pack assembler.
This adds a network dependency to evidence-pack generation. When `GITHUB_TOKEN`
is unset or the network is unavailable, `Assemble()` must complete
successfully with gap notices for review-dependent metrics. The assembler
writes the enriched pack once. No post-hoc mutation of existing packs occurs.

The workflow assembles the pack before human review. Review wait and human
review effort therefore display gap notices until the user re-runs
`shipproof evidence pack <change-id>` after review. Re-assembly rewrites
`provenance.generated_at`, which extends the measured cycle time to include
the review period. This interaction is accepted.

To associate a change with a GitHub pull request:

1. Read the `GITHUB_TOKEN` environment variable. If unset, skip all
   GitHub-dependent metrics and emit gap notices.
2. Determine the repository owner and name from the git remote URL
   (`origin` remote, parsed from HTTPS or SSH format).
3. Use the latest (most recent by timestamp) commit SHA from
   `ImplementationEvidence.commits`.
4. Call `GET /repos/{owner}/{repo}/commits/{commit_sha}/pulls`
   (Accept: `application/vnd.github.v3+json`).
5. Take the first result. If no results, emit a gap notice for
   review-dependent metrics.
6. Fetch review data from three endpoints for the matched PR:
   - `GET /repos/{owner}/{repo}/pulls/{pr}/reviews` (formal reviews)
   - `GET /repos/{owner}/{repo}/issues/{pr}/comments` (PR comments)
   - `GET /repos/{owner}/{repo}/pulls/{pr}/comments` (inline review comments)
7. Store the PR number, review timestamps, comment count, and reviewer
   list in the evidence pack as part of the single `Assemble()` write.
   Subsequent report generations read the stored data without repeating
   API calls.
8. On a 401, 403, or 429 response, log a warning and emit a gap notice
   for that change. Do not retry automatically.
9. Use a 10-second HTTP request timeout for each API call. On timeout,
   treat the call as a failure and emit a gap notice.
10. Set `per_page=100` on each list endpoint to reduce round trips.
    Fetch only the first page. Do not paginate further. This caps
    comment counts at 100 per endpoint. For most changes this is
    sufficient. The "human review effort" limitation section documents
    this cap.

## Constraints

1. The project has zero external Go dependencies. GitHub API integration
   must use `net/http` and `encoding/json` from the standard library.
   No third-party HTTP client or GitHub SDK is permitted.
2. New fields added to the EvidencePack struct (such as review metadata)
   must use `omitempty` JSON tags and must not break deserialization of
   existing packs. The schema version stays at `0.1`.
3. All HTTP requests to the GitHub API must use a 10-second timeout. The
   `http.Client` must set `Timeout: 10 * time.Second`.
4. Evidence-pack generation (`Assemble()`) must complete successfully when
   no network is available. GitHub API failures produce gap notices, not
   errors.

## Acceptance criteria

- `shipproof report project <name>` displays all ten metrics
- The five previously unavailable metrics show real derived data
- No "Unavailable Metrics" section remains in the HTML report
- Each new metric carries a provenance badge
- The five new metrics render as aggregate stat cards. Duration metrics show
  the mean across changes with data. Count metrics show sums
- Each stat card states how many changes contributed data when any change
  lacks data
- When data is missing for a metric (no PR, no shaping session), the report
  displays a specific gap notice instead of a generic "no data"
- Review-dependent metrics populate when the evidence pack is re-assembled
  after review
- Existing metrics from SP-010 (verification pass rate, first-pass success,
  agent usage, cost, requirement coverage) continue to work as before

## Scope note: project-level filtering

The project report currently aggregates all changes in the repository. The
project name is a display label only. This PRD does not add project-scoped
filtering. All ten metrics aggregate across every evidence pack under
`.shipproof/changes/`. Project membership filtering is a separate concern.

## Non-goals

- Time-series or trend analysis across report runs
- Team or individual performance rankings
- Metric thresholds or alerts
- Cross-repository aggregation
- Metrics beyond the ten listed in SDD §19
- Project-scoped change filtering (see scope note above)

## Assumptions

1. The repository is hosted on GitHub and a personal access token is
   available in the `GITHUB_TOKEN` environment variable with `repo` read
   scope during evidence-pack generation.
2. Commit SHAs in `ImplementationEvidence` can be used to find an associated
   PR. A commit belongs to at most one PR; the first match is used.
3. Git commit timestamps in `ImplementationEvidence` use RFC3339 format
   (produced by `git log --format=%aI`). The existing `cycleTimeForPack()`
   function already parses with `time.RFC3339`.

## Risks

1. If a change's commits are not in any GitHub PR, review wait and human review
   effort will be unavailable for that change. The report must show a specific
   gap notice.
2. GitHub API rate limits could slow evidence-pack generation for repositories
   with many changes. Mitigate by storing fetched review metadata in the
   evidence pack so the API is called at most once per change.

## Decomposition

Each metric is an independently verifiable change:

1. **SP-011:** Cycle time — wire up the existing `cycleTimeForPack()` function
   in `internal/report/project.go` (lines 230-279). The function already
   implements the correct logic. Remove the cycle time entry from
   `buildUnavailableMetrics()`. Add the "Delivery quality" section to the
   project report template, render the cycle time card with the mean across
   packs, and render the "Unavailable Metrics" section only when entries
   exist.
2. **SP-012:** Rework rate — derive the per-change commit count from
   `ImplementationEvidence.commits`. Emit the "No commit data available" gap
   notice when the pack has no commits. Render the rework card with the
   summed count.
3. **SP-013:** Readiness blockers — load shaping session via change record's
   `ShapingRef`. Count blockers (combining `Readiness.Blockers` and
   `Readiness.DecisionsRequired`), assumptions, risks, and unknowns. Extend
   `scanEvidencePacks()` (or add a parallel scan) to load change records from
   `.shipproof/changes/<id>/change.json` alongside evidence packs. The report
   needs the change record to resolve `ShapingRef`. Render the readiness card
   with the summed counts.
4. **SP-014:** Review data schema and API integration — add GitHub PR review
   lookup to the evidence pack assembler (`Assemble()`), extend EvidencePack
   with review metadata fields, fetch review data from three GitHub API
   endpoints. Test with `httptest.NewServer` fixtures. Cover success, 401,
   403, 429, timeout, and empty-result cases.
5. **SP-015:** Review wait and human review effort — derive from review
   metadata stored by SP-014. Render the review wait card (mean) and the
   human review effort card (sums). Removing the last entries from
   `buildUnavailableMetrics()` empties the "Unavailable Metrics" section,
   which the template hides. **Depends on SP-014.** Cannot start until
   SP-014 is complete.

Changes 1-3 require no external dependencies and can proceed first. Change 5
has a hard dependency on change 4. Changes 4-5 require GitHub API integration
and must follow in order.
