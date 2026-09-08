# SP-038 — A sample repository for the action

Status: proposed.
Source: `docs/design/shipproof-sdd.md`, Section 9, Section 14.4.
Definition of done: row P2.

## Problem

Row P2 says that the action produces a pack in continuous integration, and its
proof is a workflow run on a sample repository that uploads a pack artifact.
No sample repository exists, and no run has happened.

The action also cannot run on this repository. ShipProof holds 19 historical
changes and no open one, so `shipproof pack` with no identifier exits 2 and
names them all. A user who adds the action to a repository with no open change
meets the same failure, and a red build with no cause is worse than no build.

## Scope

Add `examples/sample-project/`, a small repository with one open change and one
passing proof. Add a `working-directory` input to the action. Add a workflow
job that runs the action against the sample and uploads the pack.

Make the action state a missing change plainly instead of failing.

## Requirements

R1. `examples/sample-project/` holds one open change with a requirement, a
proof that passes, and the code the proof judges.

R2. The action accepts a `working-directory` input. It defaults to the
repository root.

R3. The action reports a clear message and stops without failing when the
directory holds no open change.

R4. The action fails when the change exists and the pack cannot be built. A
missing pack is never silent.

R5. A workflow job runs the action against the sample repository and uploads
the pack as an artifact.

## Acceptance

- A test asserts that the sample repository holds one change and one proof.
- A test runs `pack` against the sample and asserts a `PROVEN` verdict.
- A test asserts that the action names `working-directory`.
- The workflow run on this pull request uploads a pack artifact. That run is
  the proof for row P2, and its link goes in this document.

## Non-goals

- Running the action against ShipProof itself. `AGENTS.md` pauses the
  self-hosted workflow, and the historical `.shipproof/` state stays as it is.
