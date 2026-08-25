---
name: prepare-human-review
description: Prepare a focused human-review packet for an implemented change. Use when deterministic checks and automated review are complete. Identify what a human genuinely needs to inspect and why, rather than summarizing every file.
metadata:
  shipproof-version: "0.3"
---

# Prepare human review

## Confirm the phase

```bash
shipproof next <change-id>
```

Act on the phase it reports. When it names a different skill, use that skill
first. The repository is the source of truth. Do not rely on chat history to
decide the next step.

Reduce review effort without hiding uncertainty.

Generate the evidence pack first when it does not exist:

```bash
shipproof evidence pack <change-id>
```

Generate the review packet:

```bash
shipproof review prepare <change-id>
```

The packet separates:

1. **Change intent** — what behavior changed and why.
2. **Already proven** — areas backed by deterministic evidence.
3. **Human attention required** — the smallest semantic or risk-sensitive review surface.
4. **Unknown or unproven** — anything that remains uncertain.

For each human-attention item, give:

- file or behavior;
- reason attention is required;
- relevant requirement or risk;
- evidence already available.

Do not ask humans to re-review generated files or mechanical changes when deterministic evidence already covers them.
Do not claim that omitted review areas are semantically safe without evidence.
