---
name: implement-change
description: Implement one approved ShipProof change against its intent snapshot and verification plan, then verify it. Use when coding should begin. Keep scope bounded, follow repository conventions, and never weaken verification to make the change pass.
metadata:
  shipproof-version: "0.3"
---

# Implement a change

## Before writing code

Confirm the phase:

```bash
shipproof next <change-id>
```

Act on the phase it reports. When the phase is `NO_CHANGE` or `INTENT_STALE`,
use the `prepare-change` skill first. When the phase is `NEEDS_PLAN`, use the
`plan-verification` skill first. Continue only when the phase is `NEEDS_RUN`,
`RUN_STALE`, or `RUN_FAILED`.

## Implementation

1. Read the exact intent snapshot, approved change scope, and verification plan.
2. Inspect the current implementation before editing.
3. Make the smallest coherent change that satisfies the approved scope.
4. Follow repository architecture and conventions unless the design explicitly changes them.
5. Run fast relevant checks while working.

## Verification

Execute the repository-owned verification contract:

```bash
shipproof verification run <change-id>
```

`verification run` performs two jobs. The gate runs the repository verification
command. It decides whether the repository passes. The attribution pass runs
each proof on its own. It records one result per proof in
`.shipproof/runs/<change-id>/proofs.json`.

A green attribution never masks a red gate. Read both.

Use `--gate-only` to skip the attribution pass. Use `--proofs-only` to skip
the gate. Do not use both flags together.

After the run, read `shipproof coverage <change-id>`. It states what each
requirement proved.

Confirm the intent snapshot is intact:

```bash
shipproof change check <change-id>
```

Then map results to intent:

1. Read the verification plan from `.shipproof/changes/<change-id>/verification.json`.
2. Read `.shipproof/runs/<change-id>/proofs.json`, which holds one result per
   proof. Read `.shipproof/runs/<change-id>/run.json`, which holds the gate
   result.
3. Read `shipproof coverage <change-id>`. It derives the state of each
   requirement from those two artifacts.
4. Copy the state the coverage matrix reports. Write no status the matrix does
   not report.
5. Preserve raw evidence references.

Run `shipproof next <change-id>` again. The phase advances only when the run
passes against the current revision with a clean tree.

## Hard rules

- Do not change requirements during implementation.
- Do not silently expand scope.
- Do not weaken, delete, or bypass verification to get green output.
- Do not introduce abstractions without a demonstrated need in this change.
- Record a discovered product or design conflict instead of guessing.
- A deterministic failure remains a failure.
- An unavailable check remains unknown.
- Agent explanation can interpret a result. It cannot change one.
