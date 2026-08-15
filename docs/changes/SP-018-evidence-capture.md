# SP-018 — Evidence capture profiles

## Problem

SDD §18 defines three capture profiles: metadata, redacted, and full.
The config template writes `evidence.capture: metadata`, but nothing
reads it. Telemetry collection always stores metadata only. There is no
way to capture redacted or full transcripts.

## Desired outcome

Telemetry collection enforces the configured capture level. Metadata
stores only timing and references. Redacted copies transcripts with
recognized secret shapes masked. Full copies transcripts unchanged.

## Scope

A config loader for `.shipproof/config.yaml`, an optional raw-log
provider interface on telemetry adapters, a deterministic redactor, and
capture enforcement in `telemetry collect`.

## Requirements

### SP-018-R1 — Config loader

Parse `.shipproof/config.yaml` for `verification.command` and
`evidence.capture`. Missing files default evidence capture to metadata.
Invalid capture values produce clear errors.

### SP-018-R2 — Raw log providers

Adapters can implement `RawLogProvider` to locate the most recent
session transcript. Implement it for the Claude Code and OpenCode
adapters.

### SP-018-R3 — Capture enforcement

Metadata keeps the original raw log path as a reference and copies
nothing. Redacted and full copy the transcript under
`.shipproof/runs/<change-id>/agent-raw/`. A missing raw log is not an
error; the field stays absent.

### SP-018-R4 — Redaction

A deterministic redactor masks recognized secret shapes. It must not
change ordinary text.

### SP-018-R5 — Tests

Config parsing for all capture levels and error paths. Capture
enforcement for metadata, redacted, full, missing raw logs, missing
config, and invalid config. Redactor unit tests.

## Acceptance criteria

- `just verify` passes
- Metadata capture never copies transcripts
- Redacted copies contain no recognized secret shapes
- Full copies are verbatim
- Existing telemetry collection behavior is unchanged for metadata

## Non-goals

- Prompt filtering beyond recognized secret shapes
- Per-field provider schemas

## Dependencies

SP-008 (agent telemetry).
