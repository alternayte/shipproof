---
name: triage-change
description: Assess a feature, fix, or issue and recommend the smallest useful ceremony level before work begins. Use when the work does not come from an existing design document and the right entry path is unclear.
metadata:
  shipproof-version: "0.2"
---

# Triage a change

Read the description of the work. Assess it against these criteria and recommend a ceremony level.

## Level 0 — Direct change

The work is clear enough to implement directly.

Typical indicators:

- text, validation, or configuration change;
- small bug fix with a known cause;
- bounded refactor within one area;
- low-risk dependency update.

**Next step:** use `prepare-change` to write a short change description and start the change.

## Level 1 — Shaped change

The work has unclear behavior, acceptance, or scope that a short interview can resolve.

Typical indicators:

- the desired behavior has more than one reasonable interpretation;
- acceptance criteria are not obvious;
- the implementation touches one bounded area but the approach is unclear.

**Next step:** use `shape-prd` with `shipproof shape issue` to clarify intent, then use `prepare-change`.

## Level 2 — PRD or SDD

The work is large enough that a formal document materially reduces delivery risk.

Typical indicators:

- new user workflow or system boundary;
- multiple independently deliverable changes;
- important data migration, concurrency, or security boundary;
- public API compatibility;
- significant operational risk.

**Next step:** use `shape-prd` or `shape-sdd` to produce the document, then use `decompose-plan` to break it into bounded changes.

## Level 3 — PRD plus SDD

Both product behavior and technical design contain material unresolved decisions.

**Next step:** shape both documents before decomposition.

## Rules

- Recommend the smallest level that materially reduces risk.
- State the recommendation and the one or two reasons behind it.
- The user can override the recommendation.
- Do not default to Level 2 because more ceremony feels safer.
- Do not recommend Level 0 when the acceptance criteria have more than one reasonable interpretation.
