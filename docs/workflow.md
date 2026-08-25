# ShipProof change workflow

This document describes how to implement a change with ShipProof.

## Prerequisites

Install ShipProof and initialize the repository:

```bash
go install ./cmd/shipproof
shipproof init
```

Install skills for the agent harness:

```bash
shipproof harness install claude
shipproof harness install opencode
```

## The one command you need

```bash
shipproof next <change-id>
```

`next` derives the current phase from the artifacts on disk. It names the
blocker, the exact next command, and the skill that handles it. Run it, act on
what it says, then run it again.

This document explains what each phase means. It is reference material. It is
not a sequence to remember.

| Phase | Meaning | Skill |
|---|---|---|
| `NO_CHANGE` | No change record exists | `prepare-change` |
| `INTENT_STALE` | The source document changed after the snapshot | `prepare-change` |
| `NEEDS_PLAN` | The verification plan is absent or empty | `plan-verification` |
| `NEEDS_RUN` | No run record exists | `implement-change` |
| `RUN_STALE` | The run does not describe the current tree | `implement-change` |
| `RUN_FAILED` | The newest run exited non-zero | `implement-change` |
| `NEEDS_EVIDENCE` | The run passed and no pack exists | `produce-evidence` |
| `NEEDS_REVIEW_PACKET` | The pack exists and no packet exists | `prepare-human-review` |
| `READY_FOR_HUMAN` | Every artifact is present and current | `review-change` |

## Step 1 — Prepare the change

**Skill:** `prepare-change`

The agent reads the design document (or writes a short change description for ad-hoc work), extracts the requirements for this specific change, and writes `docs/changes/<change-id>-<slug>.md`.

The agent then records the change:

```bash
shipproof change start <change-id> --source docs/changes/<change-id>-<slug>.md
shipproof change status <change-id>
```

## Step 2 — Plan verification

**Skill:** `plan-verification`

For material changes, the agent creates and populates the verification plan.

```bash
shipproof verification init <change-id>
shipproof verification check <change-id>
```

Skip this step for trivial or low-risk changes.

## Step 3 — Implement and verify

**Skill:** `implement-change`

The agent reads the intent snapshot and verification plan, then makes the smallest coherent change that satisfies the approved scope. The agent then runs the repository verification contract and confirms the intent snapshot is intact:

```bash
shipproof verification run <change-id>
shipproof change check <change-id>
```

## Step 4 — Produce evidence

**Skill:** `produce-evidence`

The agent assembles the evidence pack from intent, implementation, and verification data:

```bash
shipproof evidence pack <change-id>
```

## Step 5 — Prepare human review

**Skill:** `prepare-human-review`

The agent generates a focused review packet that separates proven areas from areas that need human attention:

```bash
shipproof review prepare <change-id>
```

## Step 6 — Generate reports

Generate human-readable reports from the evidence pack and review packet:

```bash
shipproof report change <change-id>
shipproof report pr-summary <change-id>
shipproof report project <name>
```

Use `--output <path>` to write to a file instead of stdout.

## Step 7 — Code review and commit

**Skill:** `review-change`

The agent reviews the implementation against the approved intent. After review, commit the implementation, change record, and evidence artifacts.

## Optional — Sync to Linear

When the change maps to a Linear issue or project:

```bash
shipproof linear issue <identifier>
shipproof linear project <name>
shipproof linear sync <plan-file>
```

Linear sync requires `LINEAR_API_KEY` and `LINEAR_TEAM_ID` environment variables.

## Which skill to use at each step

| Step | Skill | CLI commands |
|---|---|---|
| Prepare | `prepare-change` | `change start`, `change status` |
| Plan verification | `plan-verification` | `verification init`, `verification check` |
| Implement | `implement-change` | `change status`, `verification check`, `verification run`, `change check` |
| Evidence | `produce-evidence` | `evidence pack` |
| Human review | `prepare-human-review` | `review prepare` |
| Reports | none | `report change`, `report pr-summary`, `report project` |
| Code review | `review-change` | none |
| Linear sync | none | `linear issue`, `linear project`, `linear sync` |

## Shaping skills

Use these skills before Step 1 when the intent needs clarification:

| Skill | Purpose |
|---|---|
| `triage-change` | Assess work and recommend a ceremony level |
| `shape-prd` | Shape product intent through a bounded interview |
| `shape-sdd` | Shape technical design through a bounded interview |
| `review-prd` | Independent review of a PRD |
| `review-sdd` | Independent review of an SDD |
| `decompose-plan` | Break a large document into independently verifiable changes |
| `record-decision` | Record a durable architectural decision as an ADR |

## Starting a new session

Start a new coding-agent session when:

- the current bounded change is complete;
- the conversation carries substantial implementation history that is already in files;
- a materially different work item begins.

The repository is the source of truth. Do not rely on chat history for durable decisions.
