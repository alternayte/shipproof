---
name: produce-evidence
description: Produce a ShipProof evidence pack and concise narrative from recorded intent, implementation, verification, and review facts. Use after change checks complete. Never manufacture evidence from prose.
metadata:
  shipproof-version: "0.2"
---

# Produce evidence

## Confirm the phase

```bash
shipproof next <change-id>
```

Act on the phase it reports. When it names a different skill, use that skill
first. The repository is the source of truth. Do not rely on chat history to
decide the next step.

Build narrative from recorded evidence, never the reverse.

1. Collect the intent snapshot and requirement map.
2. Collect implementation metadata.
3. Collect deterministic verification results.
4. Collect automated review findings and human-review requirements.
5. Label each value by provenance: observed, derived, inferred, or human.
6. Keep missing values unknown.
7. Generate the evidence pack:

```bash
shipproof evidence pack <change-id> [--base <rev>] [--head <rev>]
```

8. Generate a concise summary that cites the underlying evidence identifiers.

Do not estimate missing provider cost and present it as observed.
Do not convert agent confidence into proof.
Do not hide failed runs.
