# SP-019 — Backlog reference repair

## Problem

`AGENTS.md` points to `docs/design/NEXT.md` as the ordered dogfood
backlog. That file does not exist. A fresh agent session following the
instructions hits a dead reference.

## Desired outcome

The repository instructions reference only existing documents.

## Scope

One line in `AGENTS.md`.

## Requirements

### SP-019-R1 — Reference repair

Replace the `docs/design/NEXT.md` reference with references to the SDD
for v0 goals and phases and to `docs/changes/` for the change backlog.

### SP-019-R2 — Verification

`grep -r "NEXT.md" AGENTS.md` finds no match after the change.

## Acceptance criteria

- `just verify` passes
- No document references a missing file

## Non-goals

- Recreating a backlog document

## Dependencies

None.
