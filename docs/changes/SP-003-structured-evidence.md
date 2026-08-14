# SP-003 — Structured test and static-analysis evidence

## Problem

`shipproof verification run` captures raw command output, but ShipProof cannot
extract individual test-case and static-analysis results from structured tool
output. JUnit XML and SARIF are standard formats that many tools produce. Without
parsing them, each change has exit-code evidence but no per-test or per-rule
evidence.

## Desired outcome

`internal/evidence/` parses JUnit XML and SARIF files into `schema.Check` records.
Each check carries observed provenance. A failure in the source file is a failure
in the check record, with no reinterpretation.

## Scope

Implement JUnit XML and SARIF parsers in `internal/evidence/`. Produce
`schema.Check` records from parsed results. Do not add CLI surface, write
evidence packs, or integrate with the verification runner in this change.

## Requirements

### SP-003-R1 — Parse JUnit XML

Read a JUnit XML file and extract test suites and test cases. Accept both
`<testsuite>` root and `<testsuites>` container root elements.

### SP-003-R2 — Map JUnit results to Check records

Produce one `schema.Check` per test case. Set the check ID to
`classname.name`. Map outcomes: passed test → `pass`, failed test assertion →
`fail`, test error → `fail`, skipped → `skip`. Set source to `"junit"` and
provenance to `"observed"`.

### SP-003-R3 — Parse SARIF

Read a SARIF v2.1.0 JSON file and extract run results. Each result records a
rule violation with a level (`error`, `warning`, `note`, `none`).

### SP-003-R4 — Map SARIF results to Check records

Produce one `schema.Check` per SARIF result. Set the check ID to the rule ID
when present, otherwise derive it from the result index. Map levels: `error` →
`fail`, `warning` → `unknown`, `note` → `unknown`, `none` → `skip`. Set source
to `"sarif"` and provenance to `"observed"`.

### SP-003-R5 — Never reinterpret failure

The parser must not change a failure reported by the tool into a `pass`. A test
failure, test error, or SARIF `error`-level result must produce a `fail` check.

### SP-003-R6 — Handle malformed input

Return a clear error when a file does not exist, contains invalid XML, contains
invalid JSON, or does not match the expected format.

### SP-003-R7 — Parse multiple files

Accept a list of file paths. Detect each file format and return combined
`schema.Check` records from all valid files. A file that does not match a known
format must produce an error.

## Acceptance

`go test -race ./...` must pass. Unit tests must cover parsing each format,
correct status mapping, the fail-never-becomes-pass invariant, invalid-file
errors, empty-file handling, and the combined-file function.

## Non-goals

- CLI surface for parsing.
- Writing parsed evidence to a file.
- Assembling evidence packs.
- Collecting Git evidence.
- Running parsing automatically after verification.
- Supporting formats beyond JUnit XML and SARIF v2.1.0.

## Risk

SARIF is a large specification. This change implements a focused subset
sufficient for common Go static-analysis tools such as gosec and govet. Full
SARIF conformance is deferred.
