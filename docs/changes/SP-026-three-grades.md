# SP-026 — The three provenance grades

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 6, Section 14.3.
Sequence step: Section 16, step 3, second part. It follows SP-025.

## Problem

The human report prints four grade words: `observed`, `derived`, `inferred`,
and `human`. Section 6 of the design document names three: `observed`,
`stated`, and `claimed`. A reader cannot tell `derived` from `inferred`, and
neither word states whether a tool saw the result.

The verdict rule of SP-025 reads the requirement rows. It does not read the
checks. A check that an agent asserted therefore has no defined effect on the
verdict, and row H2 has nothing to test.

## Scope

Add one package, `internal/grade`. It maps the machine label onto the three
grades of Section 6. Render the report through it. Feed the checks into the
verdict rule.

In scope:

- The three grades, and the map from every `ProvenanceKind`.
- The report badge, the report style, and the report headings.
- One sentence on the report page that states that a claimed check proves
  nothing.
- The checks in `verdict.Input`, and the rule that a claimed check never
  proves a requirement.

## Non-goals

- Any change to the `schema.ProvenanceKind` values. The machine record keeps
  the finer label, as Section 6 permits.
- Any change to `schema_version`.
- The verdict block in `prove`, in `pack`, and in the HTML report. Section 16
  step 8 covers the report rebuild.

## Requirements

R1. Only three grades exist: `observed`, `stated`, and `claimed`.

R2. `observed` maps from an observed label. `stated` maps from a human label.
`claimed` maps from a derived label and from an inferred label.

R3. An unrecognized label maps to `claimed`. A label that ShipProof does not
know is not evidence.

R4. The rendered report shows a grade word from the three only. The words
`derived` and `inferred` appear nowhere in it.

R5. The report states in words that a claimed check never proves a
requirement.

R6. The verdict rule reads the checks. A claimed check never lifts an
unproven requirement, and it never produces `PROVEN`.

R7. A failed observed check produces `FAILED`.

## Acceptance

- `go test ./internal/grade/...` passes.
- A test renders a report with a check of every label, and it asserts that
  every grade word on the page is one of three.
- A test asserts that the rendered report holds no `derived` word and no
  `inferred` word.
- A test feeds a claimed check for an unproven requirement, and it asserts
  that the verdict stays `NOT PROVEN`.
- `just verify` exits 0.

## Note on a replaced test

`TestProvenanceBadges` asserts the four old classes. Section 6 replaces that
contract. The test is rewritten to assert the three grades. This is a stricter
assertion, not a weaker one.
