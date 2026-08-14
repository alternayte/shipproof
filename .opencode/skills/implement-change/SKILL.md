---
name: implement-change
description: Implement one approved ShipProof change against its intent snapshot and verification plan. Use when coding should begin. Keep scope bounded, follow repository conventions, and never weaken verification to make the change pass.
metadata:
  shipproof-version: "0.2"
---

# Implement a change

1. Read the exact intent snapshot, approved change scope, and verification plan.
2. Inspect the existing implementation before editing.
3. Make the smallest coherent change that satisfies the approved scope.
4. Follow existing repository architecture and conventions unless the design explicitly changes them.
5. Run fast relevant checks while working.
6. Run the repository verification contract before declaring completion.

## Hard rules

- Do not change requirements during implementation.
- Do not silently expand scope.
- Do not weaken, delete, or bypass verification to get green output.
- Do not introduce abstractions without a demonstrated need in this change.
- Record a discovered product or design conflict instead of guessing.
