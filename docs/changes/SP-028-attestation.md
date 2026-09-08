# SP-028 — Sign the pack and verify the signature

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 10, Section 14.4, decision D1,
decision D2.
Sequence step: Section 16, step 5.

## Problem

A hash inside the working tree proves nothing to an auditor. The pack holds an
`attestation` section, and every pack leaves it empty. No command can tell an
altered pack from a true one.

## Scope

Add one package, `internal/attest`. It builds the canonical payload, it builds
the in-toto statement, and it verifies a signature. Add `shipproof pack
--verify <file>`.

## Decision on the cryptography

Decision D1 sets the format. Decision D2 sets the key source. Neither states
how ShipProof performs the work. The user chose this split on 2026-09-08.

- Continuous integration signs with the cosign command. Step 6 of the sequence
  adds that workflow.
- ShipProof verifies with `crypto/ecdsa` and `crypto/x509` from the standard
  library. It reads the certificate that the bundle carries.
- ShipProof adds no dependency. The repository holds none today.

ShipProof therefore checks the signature over the payload and it reports the
certificate subject. It does not check the Fulcio chain, and it does not check
the Rekor inclusion proof. `--verify` must state that limit in words. A tool
must never claim a check it did not run.

## Requirements

R1. `attest.Canonical` serialises a pack with the `attestation` field removed.
The same pack always produces the same bytes.

R2. The canonical form excludes the signature block. A signature over the
whole pack cannot exist, because the pack holds the signature.

R3. `attest.Statement` builds an in-toto statement with a SLSA provenance
predicate. The subject holds the head revision as its name and the SHA-256 of
the canonical payload as its digest.

R4. `shipproof pack --verify <file>` returns 0 for a pack whose signature
matches the canonical payload.

R5. `shipproof pack --verify <file>` returns 1 for a pack whose content
changed after the signature.

R6. `shipproof pack --verify <file>` returns 1 for an unsigned pack, and it
prints one line that states that a local pack is unsigned.

R7. `--verify` prints the checks it ran and the checks it did not run. It
names the Fulcio chain and the Rekor inclusion proof as not checked.

R8. `shipproof pack` prints one line for a local run. The line states that the
evidence is unsigned and that only a build-system pack is audit-grade.

R9. ShipProof adds no dependency.

## Acceptance

- `go test ./internal/attest/...` passes.
- A test signs a payload with a generated key, writes the pack, and asserts
  that `--verify` returns 0.
- A test alters one byte of the signed pack and asserts that `--verify`
  returns 1.
- A test asserts that `--verify` returns 1 on an unsigned pack.
- A test asserts that `go.mod` holds no require block.
- `just verify` exits 0.

## Non-goals

- The workflow that runs cosign. Step 6 of the sequence covers it.
- Fulcio chain checking and Rekor inclusion proof checking.
- Any change to the ten pack sections of Section 7.
