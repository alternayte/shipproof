# SP-034 — Any spec tool, and a full result from two commands

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 4, Section 15 decision D3,
Section 14.2.
Definition of done: rows C1 and C5.

## Problem

Two rows of the definition of done have no proof, and both fail for a real
reason.

**C1.** `start` snapshots any file correctly. It adopts no requirement from a
document that a specification tool wrote, because `ParseNative` matches only
ShipProof's own heading format, `### SP-1-R1 — statement`. An OpenSpec
proposal and a Spec Kit specification both produce an empty requirement set.
A general parser exists in `internal/requirements/foreign.go`, and nothing
calls it.

**C5.** Section 4 says that `pack` runs `prove` when no fresh result exists,
so a user reaches a full result with `start` and then `pack`. `pack` runs no
gate and no proof. A run on a clean repository writes an evidence pack and no
run record.

## Decision D3

Section 15 held D3 open. The user decided it on 2026-09-08.

**D3. DECIDED. Option A, with the confirmation gate that the code already
holds.**

ShipProof parses requirement identifiers from the intent document with one
documented pattern. A heading of level two or deeper opens a requirement, and
a list item that starts with `MUST` or `SHALL` states the obligation.

A pattern match is a proposal, never a fact. `start` writes the proposal, and
the requirements stay unadopted until a person confirms them. `Save` already
refuses an unconfirmed foreign set, and this change keeps that gate.

No adapter per specification tool exists. That would rebuild the coupling that
the strategy removes.

## Requirements

R1. `start` adopts a native requirement set when the document holds one. That
behaviour does not change.

R2. `start` proposes a requirement set when the document holds no native
requirement and the general pattern finds one.

R3. A proposal is not adopted. `start` prints each proposed requirement and
the command that confirms them.

R4. `shipproof start <change-id> --confirm-requirements` adopts the standing
proposal. It records the confirmation time.

R5. `start` reports honestly when neither the native pattern nor the general
pattern finds a requirement. It adopts nothing.

R6. `pack` runs `prove` when no proof result exists, or when the recorded
result does not describe the working tree.

R7. `pack --no-prove` skips that step, for a caller that already ran `prove`.

R8. `pack` never reports a proof result that it did not produce or read.

## Acceptance

- A test runs `start` on an OpenSpec-style document and asserts that the
  snapshot hash matches the file and that the requirements are proposed.
- A test runs `start` on a Spec Kit-style document and asserts the same.
- A test asserts that a proposal is not adopted until a person confirms it.
- A test asserts that `--confirm-requirements` adopts the proposal.
- An end-to-end test runs `start` then `pack` on a clean repository and
  asserts that the pack holds a proof result and a verdict.
- `just verify` exits 0.

## Non-goals

- An adapter per specification tool. D3 rejects it.
- Any change to the native heading format.
- The hook contract. Row T3 covers it.
