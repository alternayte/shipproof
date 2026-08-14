# SP-004 — Git evidence collector

## Problem

ShipProof can run verification and parse structured tool output, but it cannot
yet record commits, changed files, line counts, or diff metadata for a change.
Without that data, evidence packs cannot show what code was implemented.

## Desired outcome

`internal/git/` collects metadata from the local repository. It returns a
structured record with commits, changed files, addition and deletion counts, and
diff statistics. Each value carries observed provenance.

## Scope

Implement Git metadata collection in `internal/git/`. Call `git` through the
default shell. Do not add CLI surface, write evidence packs, or integrate with
the verification runner in this change.

## Requirements

### SP-004-R1 — Collect commits from a branch range

Accept two Git revision references. Return a list of commits between them. Each
commit record contains the full hash, author name, author timestamp, and commit
message subject. An empty range must return an empty list, not an error.

### SP-004-R2 — Collect changed files

Accept a revision range. Return the list of file paths that differ between the
two revisions.

### SP-004-R3 — Count additions and deletions

Accept a revision range. Return the total lines added and the total lines
deleted.

### SP-004-R4 — Collect diff stat

Accept a revision range. Return a short diff stat in the format
`<files> files changed, <insertions> insertions(+), <deletions> deletions(-)`.

This is the output from `git diff --stat <range>`.

### SP-004-R5 — Return a unified metadata record

Provide a single function that collects commits, changed files, additions,
deletions, and diff stat. Return a typed struct. Each field must carry observed
provenance.

### SP-004-R6 — Handle repository errors

Return a clear error when Git is not installed, the current directory is not a
Git repository, a revision reference does not exist, or `git` returns a
non-zero exit status.

## Acceptance

`go test -race ./...` must pass. Unit tests must cover commit collection,
file-list collection, line-count accuracy, diff-stat output, empty-range
behavior, bad-revision errors, and the unified metadata record.

## Non-goals

- CLI surface for Git evidence collection.
- Assembling evidence packs.
- Writing collected metadata to a file.
- Integrating with the verification runner.
- Detecting the base branch automatically from the repository.
- Collecting diff hunks or full file contents.
- Recording agent-run metadata.

## Risk

This change calls the `git` binary and parses its stdout. Git CLI output is
stable for the formats used here.
