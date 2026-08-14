---
name: verify-change
description: Verify a ShipProof change with deterministic repository checks and requirement mapping. Use after implementation or when evidence must be refreshed. Never reinterpret a failing deterministic check as success.
metadata:
  shipproof-version: "0.2"
---

# Verify a change

1. Read the verification plan.
2. Run the repository-owned verification entry point, normally `just verify`.
3. Run change-specific checks that are not part of the default contract.
4. Map observed results to requirements and invariants.
5. Record pass, fail, skip, or unknown exactly as observed.
6. Preserve raw evidence references.

A deterministic failure remains a failure.
An unavailable check remains unknown.
Agent explanation can interpret a result but cannot change it.
