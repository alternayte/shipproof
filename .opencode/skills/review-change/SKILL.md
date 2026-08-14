---
name: review-change
description: Review an implemented change for correctness, design, complexity, tests, maintainability, integration, and common agent-generated failure patterns. Use after deterministic verification and before human approval.
metadata:
  shipproof-version: "0.2"
---

# Review an implemented change

Review the change against its approved intent, not against an imagined broader product.

Check:

- correctness;
- design fit;
- unnecessary complexity;
- meaningful tests;
- maintainability;
- integration behavior;
- requirement drift.

Also look specifically for agent failure patterns:

- invented abstractions;
- tests that only mirror the implementation;
- unjustified dependencies;
- dead or unreachable code;
- cargo-cult defensive code;
- silent scope expansion.

Classify findings by materiality. Do not create review noise to appear thorough.
