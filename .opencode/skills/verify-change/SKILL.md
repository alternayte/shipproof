---
name: verify-change
description: Verify a ShipProof change with deterministic repository checks and requirement mapping. Use after implementation or when evidence must be refreshed. Never reinterpret a failing deterministic check as success.
metadata:
  shipproof-version: "0.2"
---

# Verify a change

## Run verification

Execute the repository-owned verification contract:

```bash
shipproof verification run <change-id>
```

Confirm the intent snapshot is intact:

```bash
shipproof change check <change-id>
```

## Map results

1. Read the verification plan from `.shipproof/changes/<change-id>/verification.json`.
2. Read the run result from `.shipproof/runs/<change-id>/run.json`.
3. Map observed results to requirements and invariants.
4. Record pass, fail, skip, or unknown exactly as observed.
5. Preserve raw evidence references.

## Rules

A deterministic failure remains a failure.
An unavailable check remains unknown.
Agent explanation can interpret a result but cannot change it.
