---
name: record-decision
description: Create or update an architecture decision record for a durable technical choice with meaningful future consequences. Do not use for reversible implementation details or routine library choices.
metadata:
  shipproof-version: "0.2"
---

# Record a technical decision

Create a concise ADR only when a decision is durable enough that future engineers need its context.

Capture:

- context;
- decision;
- rationale;
- meaningful alternatives;
- positive and negative consequences;
- status and date when available.

Do not invent rejected alternatives to make the ADR look complete.
Do not create an ADR for a choice that is cheap to reverse and locally obvious.
Use STE-assisted technical prose.
