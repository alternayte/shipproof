# SP-039 — The report explains itself

Status: implemented.
Source: `docs/changes/SP-036-comprehension-review.md`, the reader review of
2026-09-09. Section 12 and Section 2 principle 4.

## Problem

Three readers who had never used ShipProof read the change report. Every one
of them stated the verdict correctly, so row U1 is met and stays met.

Every one of them then asked what the rest of the page meant.

The verdict block was written for a stranger. The sections under it were
written for someone who already knows what ShipProof does. Seven questions
came back, and each is a reader asking the page to explain its own vocabulary.

Principle 4 of Section 2 says the result reads without training. It does for
the first three lines and not for the rest.

## Requirements

R1. The unexplained change section explains itself in one sentence before it
shows a count. A count that is not known says what would make it known.

R2. The next action explains what adding a proof means, and names the file to
edit. A reader who has never used ShipProof can act on it.

R3. The check list states what a check is and where each one came from. The
`source` column is legible without knowing the tool.

R4. The agent record states what it holds when it is present, and what it
would have held when it is absent.

R5. The page states where in the delivery flow it was produced: the revision,
and whether a build system or a person ran it.

R6. Each requirement row links to its source. A reader can see which document
the requirement came from and which line of it.

R7. The intent hash carries one sentence that says what it is for. No reader
should meet 64 characters of hexadecimal with no explanation.

R8. The verdict block does not change. It passed the review, and no
suggestion reopens a criterion that a proof met.

## Acceptance

- A test asserts that every section on the page carries a one-sentence
  explanation.
- A test asserts that the page names the source document and the revision.
- A test asserts that the requirement rows carry a source reference.
- A second reader review, with three new readers, answers the seven questions
  without asking. Record it under `docs/changes/`.
- The existing U1 record stays true: the readers still state the verdict.
- `just verify` exits 0.

## The seven findings

| # | What a reader asked | Requirement |
|---|---|---|
| 1 | What is unexplained change, and why is the count not known? | R1 |
| 2 | What does "add a proof" ask of me? | R2 |
| 3 | Which checks are these, how are they chosen, and what is `source`? | R3 |
| 4 | What would the agent record hold? | R4 |
| 5 | Where in the delivery flow does this run? | R5 |
| 6 | Where did the requirements come from? | R6 |
| 7 | What is the hash for? | R7 |

## Non-goals

- Any change to the verdict block. It works.
- Any change to the evidence pack or the schema. This is a rendering change.
- A tutorial on the page. One sentence per section, not a manual.
