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

## Entry paths

ShipProof supports two entry paths. Choose the one that fits the work.

### Path A — Change from an existing design

Use this path when a PRD, SDD, or approved design document already describes the work. The backlog contains ordered change IDs and one-line descriptions.

Start at Step 1.

### Path B — Ad-hoc feature or fix

Use this path when the work does not come from a formal design document. The change starts from a bug report, a feature idea, a user request, or an issue tracker entry.

Use the `triage-change` skill. The agent assesses the work and recommends a ceremony level:

- **Level 0** — the change is clear enough to implement directly. Use `prepare-change` to write a short change description and start at Step 1.
- **Level 1** — the change has unclear behavior or acceptance. Use `shape-prd` with `shipproof shape issue` to clarify intent through a short interview, then start at Step 1.
- **Level 2** — the change needs a PRD or SDD. Use `shape-prd` or `shape-sdd` to produce the document, then use `decompose-plan` to break it into bounded changes, then start at Step 1 for each change.
- **Level 3** — the change needs both a PRD and an SDD. Shape both documents before decomposition.

The user can override the recommendation.

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

## Step 3 — Implement

**Skill:** `implement-change`

The agent reads the intent snapshot and verification plan, then makes the smallest coherent change that satisfies the approved scope.

Before writing code, the agent confirms the change is ready:

```bash
shipproof change status <change-id>
shipproof verification check <change-id>
```

## Step 4 — Verify

**Skill:** `verify-change`

The agent runs the repository verification contract and confirms the intent snapshot is intact:

```bash
shipproof verification run <change-id>
shipproof change check <change-id>
```

## Step 5 — Produce evidence

**Skill:** `produce-evidence`

The agent assembles the evidence pack from intent, implementation, and verification data:

```bash
shipproof evidence pack <change-id>
```

## Step 6 — Prepare human review

**Skill:** `prepare-human-review`

The agent generates a focused review packet that separates proven areas from areas that need human attention:

```bash
shipproof review prepare <change-id>
```

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
| Implement | `implement-change` | `change status`, `verification check` |
| Verify | `verify-change` | `verification run`, `change check` |
| Evidence | `produce-evidence` | `evidence pack` |
| Human review | `prepare-human-review` | `review prepare` |
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
