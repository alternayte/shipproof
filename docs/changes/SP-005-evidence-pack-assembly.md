# SP-005 — Evidence-pack assembly

## Problem

SP-001 captures intent snapshots. SP-002 runs verification and writes results.
SP-003 parses structured tool output into checks. SP-004 collects Git metadata.
ShipProof cannot yet combine these sources into the versioned evidence contract
that §21 of the SDD defines.

Without assembly, each evidence source stays isolated. A reviewer cannot see one
evidence pack with intent, verification, and implementation data.

## Desired outcome

`shipproof evidence pack <change-id>` reads the intent snapshot, the
verification plan and run results, structured evidence from test and analysis
tools, and Git metadata. It assembles an `EvidencePack`, validates it against
the schema in `internal/schema/evidence.go`, and writes it to
`.shipproof/changes/<change-id>/evidence-pack.json`.

## Scope

Implement the evidence-pack assembler in a new package
`internal/evidence/pack/`. Wire it to the CLI as `shipproof evidence pack`.
Do not collect Git metadata, run verification, or parse tool output in this
change. The change only assembles existing data.

## Requirements

### SP-005-R1 — Load intent snapshot metadata

Read the change record for `<change-id>`. Extract the snapshot hash from
`sha256`. Extract requirement IDs and statements from the verification plan.

### SP-005-R2 — Load verification check results

Read the run record from `.shipproof/runs/<change-id>/run.json`. Map the exit
code to a check status: 0 to `pass`, non-zero to `fail`. Assign `observed`
provenance.

### SP-005-R3 — Load structured evidence checks

Read structured evidence from the evidence package. Parse JUnit XML and SARIF
JSON for `<change-id>` when those files exist. Convert tool output into `Check`
records with `observed` provenance.

### SP-005-R4 — Load Git evidence

Collect Git metadata using the `internal/git` package. Populate commits,
changed files, addition and deletion counts, and diff stat in the evidence
pack. Each value carries `observed` provenance.

### SP-005-R5 — Populate the EvidencePack struct

Assemble the `EvidencePack` from `internal/schema/evidence.go`. Set
`schema_version` to `"0.1"`. Set `change_id` from the argument. Fill the
`intent`, `verification`, and `provenance` fields from the loaded data.

### SP-005-R6 — Write the assembled pack

Write the pack as formatted JSON to
`.shipproof/changes/<change-id>/evidence-pack.json`. Create the directory if
it does not exist.

### SP-005-R7 — Validate the written pack

Call `EvidencePack.Validate()` after assembly and before writing. Do not write
an invalid pack.

### SP-005-R8 — Assign provenance labels correctly

Assign `observed` to values that come from automated tool output. Assign
`derived` to values that ShipProof computes. Assign `human` to
human-provided input. The pack provenance itself uses `observed`.

### SP-005-R9 — Include provenance metadata

Set `provenance.generated_at` to the current UTC timestamp in ISO 8601 format.
Set `provenance.shipproof_version` from the schema version constant
`"0.1"`.

### SP-005-R10 — CLI surface

`shipproof evidence pack <change-id>` reads all evidence sources and writes
the evidence pack. Print the output file path on success. Return a clear error
when `<change-id>` does not exist or when a required source is missing.

## Acceptance

`go test -race ./...`, `go vet ./...`, formatting checks, and `just verify`
must pass.

Unit tests must cover:
- Assembly with complete fixture inputs.
- Assembly with a missing change record.
- Assembly with a missing run record.
- Assembly with no structured evidence files.
- Pack validation (valid and invalid packs).
- Provenance label assignment.
- CLI argument validation.

## Non-goals

- Collecting Git metadata outside the `internal/git` package.
- Running verification commands.
- Parsing new tool output formats.
- Human-review packet generation.
- Agent telemetry integration.
- Incremental assembly or partial evidence packs.
- Detecting the base branch automatically. The caller must supply the revision
  range or it falls back to a sensible default.

## Risk

SP-005 reads output files from SP-001, SP-002, SP-003, and SP-004. Unit tests
must use fixture inputs that match the schema of those outputs. Integration
tests that read real output files from the dependencies are desirable but not
blocking. Assembly correctness can be verified against known fixture data.
