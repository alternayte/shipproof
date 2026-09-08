# SP-032 — Rebuild the change report around the verdict

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 12, Section 5, Section 6.
Sequence step: Section 16, step 8.

## Problem

The change report opens with an intent panel and a hash. A reader must scroll
and interpret before the report answers anything. Section 12 states the shape
the report must take, and the report does not take it.

- The verdict block does not appear on the page at all.
- The report shows no requirement table. It shows a requirement count.
- The check list is grouped by status, not by grade.
- Colour carries meaning that Section 12 does not assign. A skip pill is
  amber, and amber must mean unproven.
- The report shows an unexplained-change section that reads the same whether
  the count is zero or unmeasured.

## Scope

Rebuild `change_report.html` and the data it reads. The portfolio report stays
as it is. Section 16 step 8 names the change report, and decision D5 on the
portfolio report is still open.

## Requirements

R1. The verdict block is the first thing on the page. It holds the verdict
word, the reason line, and the next action.

R2. The verdict word carries a colour. Red means `FAILED`. Amber means
`NOT PROVEN`. Green means `PROVEN`.

R3. No other element carries a colour that means something else. A status word
uses the same three colours or no colour.

R4. The page shows one requirement table, with the requirement, its proof, its
state, and its grade.

R5. The page shows the check list grouped by grade, in the order observed,
stated, claimed.

R6. The claimed group states in words that a claimed check never proves a
requirement.

R7. The page shows the unexplained change list. An unmeasured count reads as
not known, and it names the reason.

R8. The page shows the agent record, and it states the absence when no record
exists.

R9. Every empty section states why it is empty, from `empty_sections`.

R10. The page names the signature state. An unsigned pack says so.

R11. No jargon word from the verdict word list appears in the verdict block.

R12. Only the three grades of Section 6 appear on the page.

## Acceptance

- A test asserts that the verdict block is the first content on the page.
- A test asserts one colour class per verdict, for all three verdicts.
- A test asserts that the requirement table holds the four columns.
- A test asserts that the checks are grouped by grade in order.
- A test asserts that an unmeasured unexplained count reads as not known.
- The existing H1 grade test and U2 jargon rules stay green.
- `just verify` exits 0.

## Non-goals

- The portfolio report. Decision D5 is open, and step 8 names the change
  report.
- Any change to the pack, to the schema, or to the verdict rule.
