---
name: prepare-change
description: Prepare the next ShipProof change from approved intent. Use when the next change ID is known but the scoped change description does not exist yet. Reads the SDD, PRD, or design document and writes a bounded change doc.
metadata:
  shipproof-version: "0.3"
---

# Prepare a change

## Confirm the phase

```bash
shipproof next <change-id>
```

Act on the phase it reports. When it names a different skill, use that skill
first. The repository is the source of truth. Do not rely on chat history to
decide the next step.

Write a scoped change description for one independently verifiable change.

## Workflow

1. Read the source design document (SDD, PRD, or approved intent).
2. Read `NEXT.md` or the current backlog for the change ID and its one-line description.
3. Read the existing codebase to understand what already exists.
4. Write `docs/changes/<change-id>-<slug>.md` with:
   - problem statement;
   - desired outcome;
   - scope (local to this change);
   - numbered requirements with stable identifiers;
   - acceptance criteria;
   - non-goals for this change;
   - material risk, when present.
5. Start the change record:

```bash
shipproof change start <change-id> --source docs/changes/<change-id>-<slug>.md --ceremony <level>
```

Use the level that `triage-change` recommends. Pass `--ceremony 0` for a
trivial change. Level 0 needs no verification plan. Level 1 and above need one.

6. Confirm the record:

```bash
shipproof change status <change-id>
```

## Refresh a stale snapshot

`shipproof next` reports `INTENT_STALE` when the source document changed after
the snapshot. Read the current source document first. Then re-snapshot it:

```bash
shipproof change start <change-id> --source docs/changes/<change-id>-<slug>.md --force
```

The `--force` option rewrites the record. It keeps the recorded ceremony level,
unless you also pass `--ceremony`.

## Rules

- Extract requirements from the approved design. Do not invent requirements.
- Keep the scope to one independently verifiable change.
- Do not include requirements that belong to a later change.
- Use the repository STE-assisted language profile.
- A trivial change can have a short document. Do not create ceremony for its own sake.
