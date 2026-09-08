# SP-029 — The action and the pull-request comment

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 9, Section 10, Section 14.4.
Sequence step: Section 16, step 6.

## Problem

A local pack does not satisfy an auditor. Section 9 makes the build system the
first-class path, and no workflow exists. The comment is the product for most
readers, and no command renders one.

Step 5 left two gaps that this step must close. Nothing emits the bytes that
cosign signs, and nothing writes a signature back into a pack.

## Scope

Add three options to `pack`, add one composite action, and add one workflow.
The public surface stays at five commands plus two support commands, because
every addition is an option of an existing command.

- `shipproof pack --payload <file>` prints the canonical payload. It is the
  exact byte string that a signature covers.
- `shipproof pack --attach <file> --signature <path> --certificate <path>
  --subject <rev>` writes the attestation block into a pack.
- `shipproof pack --comment <file>` prints the comment body of Section 9.2.

## Requirements

R1. `--payload` prints the bytes that `attest.Canonical` produces, and nothing
else. A signature over that output must verify.

R2. `--attach` writes a valid attestation block, and `--verify` then returns 0
for the same file.

R3. `--attach` refuses a signature that does not match the pack. It never
writes a block that `--verify` would reject.

R4. `--comment` prints the verdict block first, in the three-line shape of
Section 5.

R5. The comment holds one table with one row per requirement, its proof, and
its grade.

R6. The comment states the unexplained change count, with a reference to the
lines.

R7. The comment states the agent record when one exists, and it states the
absence when none exists.

R8. The comment ends with one sentence that names where the full pack lives.

R9. The comment holds nothing else. Section 9.2 says "this and nothing more".

R10. The action performs the four steps of Section 9.1: install the binary,
run `pack` with the merge base and the head, upload the pack as a build
artifact, and post one comment.

R11. ShipProof adds no dependency.

## Acceptance

- A test signs the output of `--payload` and asserts that `--verify` returns 0
  after `--attach`.
- A test asserts that `--attach` rejects a signature for a different pack.
- A test asserts the comment body shape: the verdict block, the requirement
  table, the unexplained line, the agent line, and the closing sentence.
- A test asserts that the comment holds no section that Section 9.2 omits.
- `just verify` exits 0.

## Non-goals

- The object store path of Section 9.3. The build artifact is the default, and
  a configured object store is a later change.
- A live workflow run. Row P2 needs a run on a sample repository, and this
  machine cannot produce that proof.
