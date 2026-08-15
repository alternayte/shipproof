# SP-017 — Intent staleness marking

## Problem

SDD §15 requires ShipProof to mark affected evidence as stale when the
source intent changes after implementation began. The CLI and the
evidence pack have no staleness signal. A changed source document silently
keeps producing evidence against outdated intent.

## Desired outcome

A deterministic comparison between the current source document and the
recorded snapshot hash marks stale intent everywhere evidence is shown.
Stale evidence needs re-verification.

## Scope

A staleness check in the change record, surfaces in `change status` and
`change check`, and a staleness field plus check in the evidence pack.

## Requirements

### SP-017-R1 — Change staleness

`Record.Staleness(root)` hashes the current source document and compares
it with the recorded SHA-256. A missing source document is stale with an
empty current hash. Other read errors are reported.

### SP-017-R2 — CLI surfaces

`change status` prints the staleness state. `change check` prints the
staleness state after hash verification without changing its exit code
semantics.

### SP-017-R3 — Evidence pack

`IntentEvidence` gains a `stale` field and an optional current source
hash. The assembler computes staleness and appends an `intent:staleness`
check with derived provenance. The check fails when the intent is stale.

### SP-017-R4 — Tests

Change tests for current, changed, and removed sources. Assembler tests
for the staleness check in both states.

## Acceptance criteria

- `just verify` passes
- Editing a source document after `change start` shows a stale state
- A stale pack carries a failing `intent:staleness` check
- A current pack carries a passing `intent:staleness` check

## Non-goals

- Automatic re-verification or evidence version chaining
- Partial-diff intent comparison

## Dependencies

SP-001 (change records and snapshots), SP-005 (evidence pack assembly).
