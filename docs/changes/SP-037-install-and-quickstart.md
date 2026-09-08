# SP-037 — An install path that needs no clone

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 2 principle 4, Section 4.

## Problem

A person who wants to use ShipProof must be able to install it and reach a
verdict without reading the source. Three things block that today.

**The quickstart needs Go.** The README offers `go install` and nothing else.
The release publishes four platform archives, and no document tells a reader
they exist.

**The README describes a product that no longer ships.** It documents
readiness states, finding classes, and four provenance labels. SP-023 removed
the first two, and SP-026 replaced the third with three grades. A newcomer
reads the wrong vocabulary before they run a command.

**The first command a newcomer runs fails.** `init` writes a default gate of
`just verify`. A repository with no `just` file then fails its gate, and the
first verdict a new user sees reads `FAILED` for a reason that has nothing to
do with their code.

## Scope

Rewrite the README opening and the quickstart. Remove the vocabulary that the
reduction cut. Make `init` choose a gate that the repository can actually run.

## Requirements

R1. The README states an install path that needs neither Go nor a clone.

R2. The quickstart reaches a verdict in numbered steps, and each step names a
command to run and what to expect.

R3. The README names no concept that SP-023 or SP-026 removed. It names three
grades, and no readiness state and no finding class.

R4. `init` detects a gate command that the repository can run. It falls back
to a command that always passes, and it says which one it chose and how to
change it.

R5. `init` never writes a gate that the repository cannot run.

## Acceptance

- A test asserts that the README holds no removed concept.
- A test asserts that the README names the release archive.
- A test asserts that `init` in a bare repository writes a runnable gate.
- A test asserts that `init` reports the gate it chose.
- `just verify` exits 0.

## Non-goals

- Publishing a release. Tagging is the owner's decision, not this change's.
- A package manager formula. The archive and `go install` cover the first
  need.
