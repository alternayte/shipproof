# ShipProof change workflow

This document describes the steps to implement one bounded change.

## Prerequisites

Build ShipProof and initialize the repository:

```bash
go build -o bin/shipproof ./cmd/shipproof
./bin/shipproof init
```

Install skills for your agent harness:

```bash
./bin/shipproof harness install claude
```

## Step 1 — Prepare the change

Use the `prepare-change` skill. The agent reads the SDD or PRD, extracts the requirements for this change, and writes the scoped change description.

The result is a document at `docs/changes/<change-id>-<slug>.md` and a recorded change with an intent snapshot:

```bash
shipproof change start <change-id> --source docs/changes/<change-id>-<slug>.md
shipproof change status <change-id>
```

Skip this step when the change description already exists.

## Step 2 — Plan verification

Use the `plan-verification` skill for material changes. The agent creates a verification plan and maps requirements to proof types.

```bash
shipproof verification init <change-id>
```

The agent populates `.shipproof/changes/<change-id>/verification.json` and validates it:

```bash
shipproof verification check <change-id>
```

Skip this step for trivial or low-risk changes.

## Step 3 — Implement

Use the `implement-change` skill. The agent reads the intent snapshot and verification plan, then makes the smallest coherent change that satisfies the approved scope.

## Step 4 — Verify

Use the `verify-change` skill. The agent runs the repository verification contract and confirms the intent snapshot is intact:

```bash
shipproof verification run <change-id>
shipproof change check <change-id>
```

## Step 5 — Review

Use the `review-change` skill for a code review against the approved intent.

Use the `prepare-human-review` skill to identify the smallest meaningful human-review surface.

## Step 6 — Commit

Commit the implementation, the change record, and any verification artifacts.

## Which skills to use

| Step | Skill | CLI commands |
|---|---|---|
| Prepare | `prepare-change` | `change start`, `change status` |
| Plan verification | `plan-verification` | `verification init`, `verification check` |
| Implement | `implement-change` | `verification run`, `change check` |
| Verify | `verify-change` | `verification run`, `change check` |
| Review | `review-change`, `prepare-human-review` | none |

## Starting a new session

Start a new coding-agent session when:

- the current bounded change is complete;
- the conversation carries substantial implementation history that is already in files;
- a materially different work item begins.

The repository is the source of truth. Do not rely on chat history for durable decisions.
