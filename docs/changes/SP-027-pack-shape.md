# SP-027 — Reshape the evidence pack to Section 7

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 7, Section 13, Section 14.2.
Sequence step: Section 16, step 4.

## Problem

Section 7 names ten top-level fields for the evidence pack. The pack on disk
holds a different set.

- `verdict` is absent. The pack states no outcome.
- `requirements` is absent. The pack holds requirement identifiers under
  `intent`, and it records no proof result for any of them.
- `checks` sits under `verification`.
- `agent` sits under `agent_run`.
- `attestation` is absent.
- `intent` records no source path and no capture time.
- Three sections that Section 7 does not name survive: `readiness`, `review`,
  and `agent_review`. The reduction removed the code that filled the first
  two.

Rule 2 of Section 7 also fails. The pack omits `unexplained_change` in
silence when no base revision is known. A reader cannot tell an absent
measurement from a measurement of zero.

## Scope

Reshape `schema.EvidencePack` to the ten fields of Section 7. Bump the schema
version. Publish the new JSON Schema. Update the assembler, the report, and
the command surface.

## Design note on rule 2

Rule 2 says that a pack states why a section is empty. Two shapes satisfy it.

- A. Every section becomes an object that carries an `absent_reason`.
- B. The sections keep the plain shape of Section 7, and one top-level
  `empty_sections` object maps a section name onto the reason.

This change takes B. It keeps every field name that Section 7 states, and it
makes the rule one testable invariant: an empty section must hold an entry in
`empty_sections`.

## Requirements

R1. The pack holds these top-level fields, and no other: `schema_version`,
`change_id`, `verdict`, `intent`, `implementation`, `requirements`, `checks`,
`unexplained_change`, `agent`, `attestation`, `empty_sections`, and
`provenance`.

R2. `verdict` holds the verdict word, the reason line, and the next action of
Section 5.

R3. `intent` holds the source path, the snapshot hash, the capture time, and
the freshness.

R4. `requirements` holds one row per requirement, with the requirement
identifier, the proof commands, the state, and the grade.

R5. `checks` is a top-level array. Every row carries a grade of Section 6.

R6. `agent` replaces `agent_run`. It is absent when no telemetry record
exists, and `empty_sections` then states why.

R7. `attestation` is present and empty for a local run. `empty_sections`
states that a local pack is unsigned.

R8. `unexplained_change` is never omitted in silence. When no base revision is
known, the section is empty and `empty_sections` states the reason.

R9. Every empty or absent section named in R1 holds an entry in
`empty_sections`.

R10. The pack is complete or absent. A failed stage writes no file.

R11. `readiness`, `review`, and `agent_review` are gone. An agent review
finding stays visible as a claimed check.

R12. `schema_version` is `0.2`, and `schemas/v0.2/evidence.schema.json`
describes the new shape.

## Acceptance

- A test asserts the exact top-level key set of a rendered pack.
- A test asserts that every empty section holds a reason in `empty_sections`.
- A test runs `pack` with a failed stage and asserts that no file exists.
- A test asserts that the pack holds no `readiness`, `review`, or
  `agent_review` key.
- The existing schema conformance test passes against `v0.2`.
- `just verify` exits 0.

## Non-goals

- The signature itself. Step 5 of Section 16 fills `attestation`.
- The verdict block in the HTML report. Step 8 covers the report rebuild.
- Any change to `change.json`, `verification.json`, or `proofs.json`.
