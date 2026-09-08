# SP-036 — The comprehension review

Status: ready to run. The result section is empty until three people read it.
Source: `docs/design/shipproof-sdd.md`, Section 14.6, row U1.
Sequence step: Section 16, step 9.

## What row U1 asks

> A reader with no ShipProof knowledge states the verdict correctly.
> Proof: a recorded review with three people who never used the tool. Record
> the result under `docs/changes/`.

This document is the packet and the record. Run the review, fill in the tables,
and the row is met or it is not.

## Who to ask

Three people who have never used ShipProof. They do not need to be engineers.
A person who reads a status report at work is the right reader.

Do not ask anyone who watched this tool get built. A reader who already knows
the vocabulary tests nothing.

## Prepare the packet

```bash
just verify
scripts/u1-packet.sh ~/u1-packet
```

That writes three pages. One shows each verdict.

| Page | Verdict | The situation it shows |
|---|---|---|
| `report-a.html` | `PROVEN` | Every requirement has a passing proof, and no line is left over. |
| `report-b.html` | `NOT PROVEN` | The proofs exist and none ran at this revision. |
| `report-c.html` | `FAILED` | A proof ran and failed. |

Show the pages in a shuffled order, and use a different order for each reader.
A reader who sees the same order learns the pattern rather than the page.

## What to say

Read this aloud, and say nothing else about the tool.

> This is a page about one change to a piece of software. I did not write it
> and I am not testing you. I want to know whether the page explains itself.
> Take as long as you like. Please think aloud if that is comfortable.

Do not define a word. Do not point at a section. If the reader asks what
something means, write the question down and say: "Whatever you think it
means."

## The questions

Ask these in order, for each page.

1. In one sentence, what is this page telling you?
2. Is this change finished and proven?
   - Choose one: **yes** / **no, not yet** / **something failed** / **I cannot
     tell**
3. What is the one thing someone should do next?
4. How many lines of this change are not explained by a requirement?
5. Was anything on the page confusing or misleading?

Question 2 is the criterion. The others explain a wrong answer.

## How to score

Question 2 maps onto the verdict.

| The reader chose | It counts as |
|---|---|
| yes | `PROVEN` |
| no, not yet | `NOT PROVEN` |
| something failed | `FAILED` |
| I cannot tell | wrong, whatever the page said |

A reader passes a page when their choice matches the verdict on that page.

**The row is met when all three readers match on all three pages.**

Nine matches out of nine. A lower score means the page did not explain itself,
and the fix belongs in the page, never in the briefing.

Question 4 has a second correct answer. On page B and page C the count is not
measured, so "it does not say" and "not known" are both right. A reader who
says "zero" got it wrong, and that is the honesty rule failing in practice.

## Record the result

Fill in one row per reader per page.

### Reader 1

Never used ShipProof: yes / no
Role, in a few words:

| Page | Verdict on the page | Reader chose | Match | Next action correct | Notes |
|---|---|---|---|---|---|
| | `PROVEN` | | | | |
| | `NOT PROVEN` | | | | |
| | `FAILED` | | | | |

### Reader 2

Never used ShipProof: yes / no
Role, in a few words:

| Page | Verdict on the page | Reader chose | Match | Next action correct | Notes |
|---|---|---|---|---|---|
| | `PROVEN` | | | | |
| | `NOT PROVEN` | | | | |
| | `FAILED` | | | | |

### Reader 3

Never used ShipProof: yes / no
Role, in a few words:

| Page | Verdict on the page | Reader chose | Match | Next action correct | Notes |
|---|---|---|---|---|---|
| | `PROVEN` | | | | |
| | `NOT PROVEN` | | | | |
| | `FAILED` | | | | |

### Score

Matches: __ of 9.

Row U1 is met: yes / no.

### Every word a reader asked about

List each one. A word that a reader had to ask about is a word the page must
lose.

-

### What to change

One line per fix. Each becomes its own change document under `docs/changes/`.

-

## A note for whoever runs this

Record what happened, not what you hoped. A reader who reached the right answer
after you explained something did not reach it. Mark that page as a miss and
write down what you had to say.
