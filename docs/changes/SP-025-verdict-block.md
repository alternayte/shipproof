# SP-025 — The verdict block

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 5, Section 14.2, Section 14.6.
Sequence step: Section 16, step 3, first part.

## Problem

ShipProof prints a phase name, a blocker, and a next command. A reader who
never used the tool cannot state the outcome from that output. The word
`NEEDS_EVIDENCE` is a machine token. It is not an answer.

Section 5 of the SDD names one output for every command that ends a flow. It
is a three-line block with a single verdict word, one count line, and one next
action. No such block exists in the code today. The proof for row C4 fails,
and the proof for row U2 has nothing to scan.

## Scope

Add one package, `internal/verdict`. It maps the state that ShipProof already
holds onto the three verdicts of Section 5. Wire it into `shipproof status`.

In scope:

- The verdict rule, from the phase, the requirement coverage, and the
  unexplained change count.
- The exact three-line render.
- A fixed jargon word list, and a test that the block excludes every word.
- The block at the head of `shipproof status`, in text form and in JSON form.

## Non-goals

- The three provenance grades of Section 6. That is a separate change, and it
  touches the report and the schema.
- The verdict block in `prove`, in `pack`, and in the HTML report. Those
  commands follow after the grades land.
- Any change to the evidence pack schema.

## Requirements

R1. Only three verdicts exist: `PROVEN`, `NOT PROVEN`, and `FAILED`.

R2. The verdict is `FAILED` when the gate ran and failed, or when a
requirement holds a failed proof.

R3. The verdict is `PROVEN` when every requirement is proven or accepted, and
the unexplained change count is a known zero.

R4. Every other state is `NOT PROVEN`.

R5. An unknown unexplained change count never produces `PROVEN`. The count is
unknown until a pack records it. The reason line states that the count is
unknown.

R6. The block holds exactly three lines. Line one holds the verdict word
alone. Line two holds the counts that drive the verdict. Line three starts
with `NEXT:` and names one runnable command.

R7. The block excludes every word on the jargon list.

R8. `shipproof status` prints the block first. The `--json` form carries the
same block under a `verdict` field.

## Acceptance

- `go test ./internal/verdict/...` passes.
- A test asserts the exact three-line shape for each verdict.
- A test asserts that an unknown unexplained count blocks `PROVEN`.
- A test scans a rendered block for each jargon word and finds none.
- A CLI test asserts that `shipproof status` prints the block first.
- `just verify` exits 0.
