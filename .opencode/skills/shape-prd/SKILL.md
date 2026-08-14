---
name: shape-prd
description: Shape a PRD through a bounded interview. Use when a product idea, feature, or problem needs enough product intent for design or implementation. Do not use for a small issue that is already clear.
metadata:
  shipproof-version: "0.2"
---

# Shape a PRD

Help the user make product intent complete enough for the next decision. Do not try to make the document perfect.

## Session state

If the work does not already have a shaping session, create one with:

```bash
shipproof shape prd "<subject>" --id <stable-id> [--source <path>]
```

Maintain `.shipproof/shaping/<stable-id>.json` as the compact decision ledger. Update it after each meaningful turn. Run `shipproof shape check <stable-id>` after edits. Do not use chat history as the only source of shaping state.

## Workflow

1. Read the shaping session and existing source material before asking questions.
2. State your current model of the problem, user, outcome, and scope.
3. Find only gaps that can materially change outcome, scope, acceptance, risk, or decomposition.
4. Ask at most three high-information questions per turn.
5. Give a recommendation when the known context supports one. Explain the trade-off briefly.
6. Record decisions, assumptions, risks, and unknowns. Do not ask the same question again after it is resolved.
7. Apply the readiness gate in [references/READINESS.md](references/READINESS.md).
8. Stop when no blocker remains. Set the session to `READY` or `READY_WITH_ASSUMPTIONS`, validate it, then produce or update the PRD.

## Rules

- Do not manufacture requirements to fill a template.
- Do not ask a question merely because more detail could be useful.
- Do not reopen an accepted decision without new evidence.
- Keep solution detail out of the PRD unless it is a real constraint.
- Recommend the smallest useful ceremony.
- Use the repository STE-assisted language profile for generated technical prose.

## Output

Maintain a compact decision ledger while shaping. When ready, produce or update the PRD and report:

- readiness state;
- blockers, if any;
- accepted assumptions;
- material risks;
- explicit deferred questions.
