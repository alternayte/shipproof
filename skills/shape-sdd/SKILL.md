---
name: shape-sdd
description: Shape an SDD through a bounded technical interview. Use when a change has material architecture, data, concurrency, API, security, migration, distributed-systems, or operational decisions. Do not require an SDD for bounded low-risk work.
metadata:
  shipproof-version: "0.2"
---

# Shape an SDD

Resolve only technical decisions that materially affect implementation or operation.

## Session state

If the work does not already have a shaping session, create one with:

```bash
shipproof shape sdd "<subject>" --id <stable-id> [--source <path>]
```

Maintain `.shipproof/shaping/<stable-id>.json` as the compact decision ledger. Update it after each meaningful turn. Run `shipproof shape check <stable-id>` after edits. Do not use chat history as the only source of shaping state.

## Before asking questions

Read, when available:

- approved product intent;
- relevant source code;
- repository instructions;
- existing ADRs and nearby designs;
- deployment and infrastructure constraints.

Do not ask the user to restate facts the repository already answers.

## Workflow

1. State your current system model and recommended design direction.
2. Explain why that direction fits existing constraints.
3. Find missing information that can materially change architecture, data ownership, interfaces, invariants, failure behavior, rollout, or verification.
4. Ask at most three high-information questions per turn.
5. Prefer the simplest design that satisfies known requirements and appetite.
6. Apply only the relevant lenses in [references/SDD-LENSES.md](references/SDD-LENSES.md).
7. Record decisions, assumptions, risks, and unknowns.
8. Apply [references/READINESS.md](references/READINESS.md).
9. Stop when no blocker remains. Set the session to `READY` or `READY_WITH_ASSUMPTIONS`, validate it, then produce or update the SDD.

## Rules

- Reject cargo-cult architecture.
- Do not introduce infrastructure only because it is fashionable.
- Do not require alternatives when no real trade-off exists.
- Do not invent NFR targets.
- Mark missing information as unknown.
- Use the STE-assisted language profile for generated technical prose.
