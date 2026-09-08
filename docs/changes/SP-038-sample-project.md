# SP-038 — A sample repository for the action

Status: implemented. Row P2 is met.
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
  the proof for row P2.

## The proof for row P2

Run 34284905727 on pull request 1.
https://github.com/alternayte/shipproof/actions/runs/34284905727

It uploaded `shipproof-evidence-pack`, 4284 bytes. The pack reads:

```
schema_version: 0.3
verdict:        PROVEN
signed:         true
subject:        20fc193acc8ae8de791eb5c26647824b7d9995e3
```

The build system signed it with keyless Sigstore, and the local binary
verifies that signature:

```
$ shipproof pack --verify evidence-pack.json
SIGNATURE: VALID
subject   20fc193acc8ae8de791eb5c26647824b7d9995e3
digest    sha256:0701ab6611c9a383a82ced110aa57da4d2bf6b77ea6f9452b272fd8855c36a3a
signer    https://github.com/alternayte/shipproof/.github/workflows/evidence.yml@refs/pull/1/merge
checked   the signature over the canonical payload
checked   the digest of the canonical payload
unchecked the Fulcio certificate chain
unchecked the Rekor inclusion proof
exit=0
```

The action also posted one comment with the verdict block, which is the
proof for row P3 against a live run rather than a test.

## Two defects the live run found

The first run failed. Cosign signed the pack and the verifier then refused
its own signature, because `cosign sign-blob` base64-encodes its output and
the verifier read only raw PEM. No unit test could have found it. The bug
lived in the space between two tools.

The second was quieter. The signed pack still listed `attestation` under
`empty_sections`, so it carried a signature and called that section empty.
`Validate` now rejects a filled section that still holds a reason, and the
canonical payload drops that entry, so signing never changes the bytes the
signature covers.

## Non-goals

- Running the action against ShipProof itself. `AGENTS.md` pauses the
  self-hosted workflow, and the historical `.shipproof/` state stays as it is.
