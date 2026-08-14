---
name: review-sdd
description: Review an SDD independently for correctness, material design gaps, and readiness. Use after an SDD draft exists. Apply only relevant technical lenses and distinguish blockers from optional improvements.
metadata:
  shipproof-version: "0.2"
---

# Review an SDD

Determine whether the design is coherent enough to implement responsibly.

Read [references/SDD-LENSES.md](references/SDD-LENSES.md) for conditional review lenses.

## Method

1. Trace the design back to approved intent.
2. Identify affected boundaries, interfaces, ownership, invariants, and material decisions.
3. Apply only lenses that match the change.
4. Challenge unjustified complexity and missing failure semantics.
5. Classify every finding as `BLOCKER`, `DECISION`, `ASSUMPTION`, `RISK`, `SUGGESTION`, or `NIT`.
6. Never let `SUGGESTION` or `NIT` block readiness.
7. If no blocker or unresolved decision remains, declare readiness and stop.

Do not turn the review into a checklist-completeness exercise.
