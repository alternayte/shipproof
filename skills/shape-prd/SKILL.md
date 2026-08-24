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

## Stop rule

Apply this test before you write another question or another blocker.

1. Check the readiness gate in [references/READINESS.md](references/READINESS.md).
2. For each remaining gap, ask one question: does the answer change the outcome, the scope, the acceptance, the architecture, the risk, or the decomposition?
3. If the answer is no, the gap is a `SUGGESTION` or a `NIT`. Record it. Do not block on it.
4. If no `BLOCKER` and no unresolved `DECISION` remains, stop and declare `READY` or `READY_WITH_ASSUMPTIONS`.
5. Declare `READY_WITH_ASSUMPTIONS` when you recorded an assumption. Declare `READY` when you did not.

Stop even when the document can still improve. A model can always propose one more question. A remaining suggestion is not a reason to continue.

### Optional detail never blocks

These are optional details. Record them as suggestions or assumptions. Never make one a blocker:

- an edge case that does not change the main outcome;
- a metric threshold that the team can set later;
- a rollout date, a rollout order, or a communication plan;
- an error message, a label, or wording;
- a field name, a schema detail, or an implementation choice;
- a generic non-functional requirement with no contextual source;
- a permission, a limit, or a default that a stated constraint already implies.

## Rules

- Do not manufacture requirements to fill a template.
- Do not ask a question merely because more detail could be useful.
- Do not declare `BLOCKED` for a detail that the stop rule classifies as optional.
- Do not ask a fourth question in one turn.
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
