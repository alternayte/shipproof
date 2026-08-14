---
name: decompose-plan
description: Decompose approved intent and design into independently verifiable delivery changes. Use when a PRD, SDD, RFC, or larger feature needs Linear-ready issues and dependencies. Do not invent work not justified by the source.
metadata:
  shipproof-version: "0.2"
---

# Decompose a delivery plan

Turn approved intent into the smallest coherent set of independently verifiable changes.

## Workflow

1. Read the approved intent and readiness state.
2. Refuse decomposition when a blocker remains.
3. Extract stable requirements and material constraints.
4. Prefer vertical slices that produce observable behavior.
5. Create a change only when it can be implemented and verified coherently.
6. Add dependencies only when one change truly blocks another.
7. Map each change to the requirements it satisfies.
8. Flag uncovered requirements.
9. Flag changes that are too broad to review or verify well.

Do not invent milestones, epics, or dependencies for visual symmetry.
Do not split mechanical implementation layers into separate issues when one vertical change is clearer.
