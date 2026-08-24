# Complete project report metrics — Software Design Document

> **Not canonical.** This work is complete. The document stays as a record of
> changes SP-011 to SP-014. `docs/design/shipproof-v0-sdd.md` is the contract
> for v0. This document creates no requirement.

**Status:** ready_with_assumptions
**Date:** 2026-08-15
**Shaping session:** `.shipproof/shaping/complete-metrics-sdd.json`
**Source:** `docs/prd/complete-metrics.md`

## 1. Context

The project aggregate report derives five metrics from evidence packs.
Five more metrics show hard-coded "no data available" placeholders.

This design makes all ten metrics available:

- cycle time;
- review wait;
- verification pass rate;
- first-pass success;
- rework;
- agent usage;
- cost;
- human review effort;
- requirement coverage;
- readiness blockers.

Three metrics derive from data already on disk. Two metrics need GitHub PR
data. One new package and one new command collect that data. The evidence
pack schema gains two optional fields. The report derives all new metrics
from pack data, which keeps report generation offline and deterministic.

## 2. Design constraints

The design must follow existing repository patterns:

- evidence pack assembly is offline and deterministic;
- the schema validates strictly against version 0.1;
- external API integrations are opt-in commands that sync data to disk files,
  per the Linear adapter;
- reports derive everything from evidence packs on disk;
- every derived metric carries a provenance badge.

## 3. Schema extension

Add two optional fields to `EvidencePack` in `internal/schema/evidence.go`.
The schema version stays 0.1. Existing packs stay valid.

### 3.1 ReadinessEvidence

```go
type ReadinessEvidence struct {
    ShapingRef   string `json:"shaping_ref,omitempty"`
    BlockerCount int    `json:"blocker_count,omitempty"`
}
```

Top-level field `Readiness *ReadinessEvidence` with `json:"readiness,omitempty"`.

A nil pointer means no shaping data. The report shows zero blockers for that
change. The shaping ref lets a reader verify the count against the shaping
session file.

### 3.2 ReviewEvidence

```go
type ReviewEvidence struct {
    Source            string `json:"source"`
    PRNumber          int    `json:"pr_number"`
    PRURL             string `json:"pr_url"`
    OpenedAt          string `json:"opened_at,omitempty"`
    FirstReviewAt     string `json:"first_review_at,omitempty"`
    ReviewCount       int    `json:"review_count,omitempty"`
    CommentCount      int    `json:"comment_count,omitempty"`
    DistinctReviewers int    `json:"distinct_reviewers,omitempty"`
    State             string `json:"state,omitempty"`
    CollectedAt       string `json:"collected_at"`
}
```

Top-level field `Review *ReviewEvidence` with `json:"review,omitempty"`.

The pack stores raw facts. Review wait and human review effort are derived at
report time, like metrics R2 through R6.

Validation stays strict for existing fields. New fields validate their
timestamps as RFC 3339 when present.

## 4. GitHub client package

Create `internal/github` with the same shape as `internal/linear`:

- `Client` with token and base URL;
- `NewClient(token)` and `NewClientWithURL(token, baseURL)` for tests;
- `GITHUB_TOKEN` env var resolution;
- typed errors: no token, auth failed, rate limit, not found;
- one GraphQL query.

### 4.1 Query

The query finds the PR for a commit SHA:

```graphql
query($owner: String!, $name: String!, $sha: String!) {
  repository(owner: $owner, name: $name) {
    object(expression: $sha) {
      ... on Commit {
        associatedPullRequests(first: 1) {
          nodes {
            number
            url
            createdAt
            state
            reviews(last: 100) {
              totalCount
              nodes {
                submittedAt
                author { login }
              }
            }
            reviewThreads(first: 1) { totalCount }
          }
        }
      }
    }
  }
}
```

`reviews(last: 100)` returns the oldest reviews first. The first node gives
the earliest review time. More than 100 reviews per PR is out of scope.

### 4.2 Repository resolution

Add a helper to `internal/git` that parses the origin remote URL:

```go
func ResolveGitHubRepo(dir string) (owner, name string, err error)
```

It runs `git remote get-url origin` and parses both URL forms:

- `https://github.com/<owner>/<name>.git`
- `git@github.com:<owner>/<name>.git`

A missing origin or a non-GitHub origin returns a clear error.

## 5. Evidence review command

Add `shipproof evidence review <change-id>` to the CLI.

### 5.1 Steps

1. Load the evidence pack for the change. Error with "run shipproof evidence
   pack <id> first" when the pack does not exist.
2. Resolve `GITHUB_TOKEN`. Error when missing.
3. Resolve the repo owner and name from the origin remote.
4. Try each commit SHA from `ImplementationEvidence.commits` in order.
   Stop at the first SHA that yields a PR. First match wins.
5. Build `ReviewEvidence` from the PR data. Set `CollectedAt` to now.
6. Write `review.json` to `.shipproof/changes/<change-id>/`.
7. Print a short summary to stdout.

### 5.2 Exit codes and failures

- 2 for usage errors;
- 1 for operational errors: no pack, no token, no PR found, API errors;
- 0 on success.

No PR found means the change's commits are not in any GitHub PR. The command
reports this clearly. The pack keeps no review data. The report shows a gap
notice for review wait and human review effort.

## 6. Assembler changes

In `internal/evidence/pack/assembler.go`:

### 6.1 Readiness snapshot

After loading the change record:

- when `record.ShapingRef` is empty, skip;
- load the shaping session with `shaping.Load(root, ref)`;
- set `pack.Readiness` with the ref and `len(session.Readiness.Blockers)`;
- when the session file is missing, skip silently. The report shows zero.

### 6.2 Review merge

After loading the agent run:

- read `.shipproof/changes/<change-id>/review.json`;
- when the file is missing, skip silently;
- when it exists, parse and set `pack.Review`;
- append a check `github:review` with status `pass`, source `github`, and
  observed provenance, matching the `git:collect` pattern;
- a malformed `review.json` returns an error. The file is deterministic
  output from the evidence review command.

## 7. Report derivation

In `internal/report/project.go`:

### 7.1 Cycle time

Per change: oldest commit timestamp to `provenance.generated_at`.
The existing uncommitted `cycleTimeForPack` function implements this.
Wire it into the metrics. Compute the project average across changes with
valid values. Gap notice when commit data is missing or timestamps do not
parse.

### 7.2 Rework

Per change: count of `ImplementationEvidence.commits`.
Project aggregate: average commits per change.

### 7.3 Readiness blockers

Per change: `pack.Readiness.BlockerCount` when present, else zero.
Project aggregate: total blockers across changes.

### 7.4 Review wait

Per change:

- start: `pack.AgentRun.EndedAt` when present, else the latest commit
  timestamp;
- end: `pack.Review.FirstReviewAt`;
- gap notice when `pack.Review` is nil, when `FirstReviewAt` is empty, when
  timestamps do not parse, or when end is before start.

Project aggregate: average across changes with valid values.

### 7.5 Human review effort

Per change: `ReviewCount`, `CommentCount`, `DistinctReviewers`.
Project aggregate: totals across changes, and distinct reviewers across all
changes.

## 8. Template changes

In `internal/report/templates/project_report.html`:

- remove the unavailable metrics section and its styles;
- add new metric cards with derived provenance badges for cycle time,
  rework, review wait, human review effort, and readiness blockers;
- extend the pack summary table with per-change columns: cycle time,
  commits, review wait, review comments, reviewers, blockers;
- keep gap notices visible in per-change rows.

## 9. Decomposition

Four bounded changes. One depends on another.

### SP-011 — Cycle time and rework metrics

Pure report derivation. No schema change. Incorporate the uncommitted
`cycleTimeForPack` WIP. Wire both metrics into the template. Remove "Cycle
time" and "Rework rate" from the unavailable list.

### SP-012 — Readiness blocker snapshot

Schema `ReadinessEvidence`, assembler embedding via `ShapingRef`, report
rendering. Incorporate the uncommitted `ShapingRef` WIP in `internal/change`
and the CLI `--shaping` flag. Remove "Readiness blockers" from the
unavailable list.

### SP-013 — GitHub review client and command

`internal/github` package, git remote resolution helper, `shipproof evidence
review` command, `review.json` output. No report changes yet.

### SP-014 — Review metrics end-to-end

Schema `ReviewEvidence`, assembler merge with `github:review` check, report
derivation of review wait and human review effort, template cards and
columns. Remove the unavailable section entirely.

SP-014 depends on SP-013. SP-011 and SP-012 are independent.

## 10. Verification approach

Each change gets deterministic checks:

- unit tests for derivation functions with fixture packs, following
  `internal/report/project_test.go`;
- `internal/github` client tests with `httptest`, following
  `internal/linear/client_test.go`;
- CLI tests with temp roots, following `internal/cli/evidence_test.go`;
- template content assertions for the new metric sections;
- schema validation tests for the new optional fields;
- `just verify` must pass.

## 11. Assumptions

1. The GitHub reviews query caps at 100 reviews per PR. The first node of
   `reviews(last: 100)` is the earliest review.
2. Review comment count means `reviews.totalCount` plus
   `reviewThreads.totalCount` from the GitHub GraphQL API.

## 12. Risks

1. Uncommitted WIP in `project.go` and `change.go` must be incorporated into
   SP-011 and SP-012.
2. GitHub API rate limits could block evidence review for many changes. The
   on-disk `review.json` acts as a cache: one fetch per change, none at
   report time.
3. A commit may appear in multiple PRs or none. First match wins. No match
   fails the command clearly.
4. New checks change the verification pass rate totals. This is accepted:
   the checks are observed data with provenance labels.

## 13. Non-goals

- PR review comments per thread or per line;
- pagination beyond 100 reviews;
- GitHub Enterprise URLs;
- refresh of review data at report time;
- storing cycle time or review wait values in the pack. Reports derive them.
