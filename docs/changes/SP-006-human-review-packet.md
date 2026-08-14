# SP-006 — Human-review packet

## Problem

SP-005 assembles an evidence pack with intent, verification, implementation, and
provenance data. The pack includes every check with every provenance label.
A human reviewer must manually inspect the pack to find items that need
attention. There is no deterministic filter that separates checks that are
already proven from checks that need human interpretation.

Without a focused review packet, the human must read the full evidence pack and
decide what to inspect. This causes wasted effort and makes it easy to miss
findings that require attention.

## Desired outcome

`shipproof review prepare <change-id>` reads the evidence pack for
`<change-id>` and writes a focused review packet. The packet groups checks into
four sections per the `prepare-human-review` skill: change intent, already
proven, human attention required, and unknown or unproven. It excludes checks
with `observed` provenance that passed. It identifies items with `inferred`
provenance or `fail` status for human inspection.

## Scope

Implement the review-packet generator in a new package `internal/review/`. Wire
it to the CLI as `shipproof review prepare`. The change reads the evidence pack
from SP-005 and applies the deterministic filtering rules from the
`prepare-human-review` skill. Do not run verification or collect new evidence in
this change.

## Requirements

### SP-006-R1 — Load the evidence pack

Read `evidence-pack.json` from `.shipproof/changes/<change-id>/`. Return a clear
error when the pack does not exist or cannot be parsed.

### SP-006-R2 — Build the change intent section

Extract the change ID and intent summary from the evidence pack. Include the
snapshot hash and a count of requirements.

### SP-006-R3 — Build the already proven section

List verification checks with `observed` provenance and `pass` status. For each
check, include the check ID, source, and status.

### SP-006-R4 — Build the human attention section

List checks with `inferred` provenance or `fail` status. For each item, include
the check ID, status, provenance, source, the reason attention is required, the
relevant requirement IDs, and the evidence already available.

### SP-006-R5 — Build the unknown or unproven section

List checks with `unknown` status or with no matching requirement. For each
item, include the check ID, status, provenance, and what is uncertain.

### SP-006-R6 — Exclude observed-pass checks from human attention

Do not include checks with `observed` provenance and `pass` status in the human
attention or unknown sections. These checks are already proven and require no
human inspection.

### SP-006-R7 — Include the Git implementation summary

Copy the commit list, changed file count, additions, and deletions from the
evidence pack into the review packet under a Git summary.

### SP-006-R8 — Write the review packet

Write the packet as formatted JSON to
`.shipproof/changes/<change-id>/review-packet.json`. Create the directory if it
does not exist.

### SP-006-R9 — CLI surface

`shipproof review prepare <change-id>` reads the evidence pack and writes the
review packet. Print the output file path on success. Return a clear error when
`<change-id>` does not exist or when the evidence pack is missing.

## Acceptance

`go test -race ./...`, `go vet ./...`, formatting checks, and `just verify`
must pass.

Unit tests must cover:
- Packet generation with a complete evidence pack.
- Packet generation with a mix of pass, fail, and inferred checks.
- Packet generation with all observed-pass checks.
- Packet generation with a missing evidence pack.
- Packet generation with a missing change record.
- Correct section assignment per the provenance and status rules.
- CLI argument validation.

## Non-goals

- Running verification commands.
- Collecting new Git or implementation evidence.
- Generating human-readable prose summaries.
- Analyzing diff content for semantic risk.
- Agent telemetry integration.
- Generating the evidence pack.

## Risk

SP-006 depends on the evidence pack schema from SP-005. If the evidence pack
schema changes, the review-packet generator must be updated. Unit tests must use
fixture evidence packs that match the current schema.
