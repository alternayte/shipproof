# SP-030 — The control mapping document

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 11, Section 13, Section 14.4.
Sequence step: Section 16, step 7.

## Problem

An enterprise buyer does not buy a check identifier. The buyer buys an answer
to an audit request. Section 11 names one document, `docs/controls.md`, and it
does not exist. No test binds the document to the schema, so the document can
drift and no proof reports the drift.

Section 11 also names one field that the pack does not carry. It names
`checks[].grade`. A check row holds `provenance`, which carries four values. A
reader must therefore know the collapse rule of Section 6 to answer the audit
question "Is the result machine-observed or asserted?".

## Scope

Add `docs/controls.md`. Add a test that binds every field the document names
to the schema. Add `grade` to a check row.

## Decision on the missing field

Section 11 says that each pack field maps to a named control expectation. The
pack must therefore answer the audit question on its own. A pack that needs an
external lookup table does not answer it.

Two shapes satisfy the section.

- A. `docs/controls.md` names `checks[].provenance`, and it states the collapse
  rule beside the row. No schema change.
- B. A check row carries `grade` beside `provenance`. The pack answers the
  question directly. `provenance` stays as the finer machine label that
  Section 6 permits.

This change takes B. Section 11 names the field, and an auditor reads the pack,
not the source. The addition is additive, and Section 13 requires a
`schema_version` change for it.

## Requirements

R1. `docs/controls.md` exists.

R2. Every pack field that the document names exists in the schema.

R3. The document holds one row per audit question of Section 11.

R4. The document maps each row to SOC 2 change management, to EU AI Act record
keeping, and to SOX change control.

R5. The document states the mapping as a claim about evidence. It never states
a certification. It says so in words.

R6. A check row carries `grade`. The value is one of the three grades of
Section 6.

R7. `provenance` stays on a check row. It is the finer machine label.

R8. `schema_version` is `0.3`, and `schemas/v0.3/evidence.schema.json`
describes the new shape.

R9. A recorded pack from an older version still answers to the schema of its
own version.

## Acceptance

- A test reads every backtick-quoted pack field from `docs/controls.md` and
  asserts that the schema holds it.
- A test asserts that the document names every audit question of Section 11.
- A test asserts that the document holds no certification claim.
- A test asserts that an assembled check row carries a grade from the three.
- `just verify` exits 0.

## Non-goals

- Any control framework beyond the three that Section 11 names.
- A per-framework report. Section 11 names one document and one command
  output, and the command output is the pack itself.
