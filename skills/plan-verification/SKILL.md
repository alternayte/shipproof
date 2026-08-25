---
name: plan-verification
description: Plan how to prove a change before implementation. Use for material changes with explicit requirements, invariants, failure behavior, or risk. Prefer deterministic evidence and mark human-only checks explicitly.
metadata:
  shipproof-version: "0.2"
---

# Plan verification

## Confirm the phase

```bash
shipproof next <change-id>
```

Act on the phase it reports. When it names a different skill, use that skill
first. The repository is the source of truth. Do not rely on chat history to
decide the next step.

Map important intent to proof before implementation details bias the tests.

Create the repository-owned plan first when it does not exist:

```bash
shipproof verification init <change-id>
```

Maintain `.shipproof/changes/<change-id>/verification.json`. Run `shipproof verification check <change-id>` before implementation starts.

For each material requirement or invariant:

1. State what must be proven.
2. Choose the cheapest reliable proof type.
3. Prefer deterministic checks: unit, integration, contract, property, migration, load, static, or end-to-end tests.
4. Mark a check as human-only when automation would not prove the behavior responsibly.
5. Identify hidden failure cases that an implementation-shaped test could miss.

Do not invent acceptance criteria.
Do not weaken a requirement because it is hard to test.
Do not require an elaborate verification plan for a trivial low-risk change.
