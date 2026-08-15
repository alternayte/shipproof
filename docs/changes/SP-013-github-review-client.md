# SP-013 — GitHub review client and command

## Problem

Review wait and human review effort need GitHub PR review data. ShipProof
has no GitHub client and no command that collects PR data. The Linear
adapter established the pattern: an opt-in command syncs external data to a
disk file.

## Desired outcome

`shipproof evidence review <change-id>` finds the GitHub PR for the
change's commits and writes `review.json` to the change directory. The
command is opt-in. Evidence pack assembly stays offline.

## Scope

Create `internal/github` with a GraphQL client. Add a git helper that
parses the origin remote URL. Add the CLI command. No report changes.

## Requirements

### SP-013-R1 — GitHub client

`internal/github` follows the Linear client shape: `Client` with token and
base URL, `NewClient`, `NewClientWithURL`, `GITHUB_TOKEN` resolution, and
typed errors for no token, auth failure, rate limit, and not found.

### SP-013-R2 — PR lookup query

GraphQL query per SDD §4.1: for a commit SHA, return the associated PR
with number, URL, created date, state, the earliest 100 submitted reviews
with authors, and review thread count.

### SP-013-R3 — Repository resolution

`git.ResolveGitHubRepo(dir)` parses `git remote get-url origin` for HTTPS
and SSH GitHub URL forms. Missing or non-GitHub origin returns a clear
error.

### SP-013-R4 — evidence review command

`shipproof evidence review <change-id>`:

1. Loads the evidence pack; errors with "run shipproof evidence pack
   <id> first" when missing.
2. Resolves `GITHUB_TOKEN`; errors when missing.
3. Resolves repo owner and name from the origin remote.
4. Tries each commit SHA in implementation order; first PR match wins.
5. Writes `review.json` to `.shipproof/changes/<change-id>/` using the
   `ReviewEvidence` JSON shape from SDD §3.2.
6. Prints a short summary.

Exit codes: 2 for usage, 1 for operational errors, 0 on success.

### SP-013-R5 — Tests

Client tests with `httptest`, following `linear/client_test.go`. Remote
URL parsing tests. CLI tests with temp roots and a stub server.

## Acceptance criteria

- `just verify` passes
- The command writes valid `review.json` for a change whose commits are in
  a PR
- No PR found and missing token produce clear errors with exit code 1
- Evidence pack assembly remains unchanged by this change

## Non-goals

- Report rendering of review metrics (SP-014)
- Assembler merge of `review.json` (SP-014)
- GitHub Enterprise URLs

## Dependencies

None.
