---
name: review-prd
description: Review a PRD independently for material product-intent defects and readiness. Use after a PRD draft exists. Do not use to endlessly improve wording or reopen accepted choices without evidence.
metadata:
  shipproof-version: "0.2"
---

# Review a PRD

Review whether the PRD is complete enough for design or implementation.

Read [references/PRD-RUBRIC.md](references/PRD-RUBRIC.md) when you need detailed checks.

## Method

1. Identify the problem, actor, desired outcome, scope, appetite, key behavior, acceptance, dependencies, assumptions, and risks.
2. Report only findings that fit a ShipProof class: `BLOCKER`, `DECISION`, `ASSUMPTION`, `RISK`, `SUGGESTION`, or `NIT`.
3. Treat `SUGGESTION` and `NIT` as non-blocking.
4. Do not create blockers from hypothetical future needs.
5. Do not require sections that are not relevant.
6. If no blocker or unresolved decision remains, declare `READY` or `READY_WITH_ASSUMPTIONS` and stop.

## Review standard

Ask: "Can the next stage make responsible decisions from this intent?"

Do not ask: "Can I find one more thing to criticize?"
