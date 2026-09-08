# SP-031 — Grade a ShipProof measurement as observed

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 5, Section 6, Section 11.
Follows SP-030. It precedes step 8 of Section 16.

## Problem

SP-030 put `grade` on every check row, and `docs/controls.md` now points an
auditor at that field to answer "Is the result machine-observed or asserted?".
The field is load-bearing, and two of its values are wrong.

The `intent:staleness` check reads two files, computes a SHA-256 for each, and
compares them. It carries the label `derived`, which Section 6 collapses into
`claimed`. A pack therefore tells an auditor that an agent asserted the
freshness of the intent. No agent did. A machine measured it.

A coverage row that nothing proved carries the same wrong label. ShipProof read
the recorded results and found none. That absence is a measurement, not a
claim.

The correction exposes a second defect. The verdict rule treats any failed
observed check as `FAILED`. A stale intent would then read `FAILED`. Section 5
reserves that word: "A proof ran and failed, or the gate failed." A stale
intent is neither. It is `NOT PROVEN`.

## The rule this change writes down

A check is `observed` when a machine measured its status and can repeat the
measurement. A file hash, an exit code, and a parsed tool report all qualify.
ShipProof is such a machine, so a comparison that ShipProof performs is a
measurement.

A check is `claimed` when its status rests on a natural-language assertion. An
agent review finding stays claimed.

## Scope

Correct the two labels. Narrow the `FAILED` rule to what Section 5 states.

## Requirements

R1. The `intent:staleness` check carries the observed label.

R2. A coverage check for a requirement with no recorded result carries the
observed label. ShipProof measured the absence.

R3. An agent review finding stays claimed. It rests on a summary sentence.

R4. `FAILED` follows only from a failed proof or a failed gate. A check that
reports the state of the pack never produces it.

R5. A stale intent produces `NOT PROVEN`. It never produces `FAILED`.

R6. A failing test report and a failing gate still produce `FAILED`. This
change must not hide a real failure.

R7. No schema change. The values are already in the vocabulary.

## Acceptance

- A test asserts that an assembled `intent:staleness` check grades observed.
- A test asserts that a coverage check with no result grades observed.
- A test asserts that an agent review check still grades claimed.
- A test asserts that a stale intent produces `NOT PROVEN`.
- A test asserts that a failed test report still produces `FAILED`.
- `just verify` exits 0.

## A test this change replaces

`TestAFailedObservedCheckFails` asserts that any failed observed check
produces `FAILED`. SP-026 wrote it, and it reaches wider than Section 5
states. The replacement asserts the Section 5 rule: a failed proof and a
failed gate produce `FAILED`, and a failed informational check does not.

This narrows one assertion and adds two. It is not a weaker test suite. It is
a suite that matches the design document.

## Non-goals

- Any change to the four machine labels under `checks[].provenance`.
- Any change to the report or to the comment.
