# ShipProof change workflow

This document describes how to implement a change with ShipProof.

## Prerequisites

Install ShipProof and initialize the repository:

```bash
go install ./cmd/shipproof
shipproof init
```

`init` also installs the ShipProof instructions into every harness path.

## The one command you need

```bash
shipproof status <change-id>
```

`status` derives the current phase from the artifacts on disk. It names the
blocker, the exact next command, the skill that handles it, and the
requirement coverage. Run it. Act on what it says. Run it again.

This document explains what each phase means. It is reference material. It is
not a sequence to remember.

| Phase | Meaning | Skill |
|---|---|---|
| `NO_CHANGE` | No change record exists | `capture-intent` |
| `INTENT_STALE` | The source document changed after the snapshot | `capture-intent` |
| `NEEDS_PLAN` | The verification plan is absent or empty | `plan-proof` |
| `NEEDS_RUN` | No run record exists | `plan-proof` |
| `RUN_STALE` | The run does not describe the current tree | `plan-proof` |
| `RUN_FAILED` | The newest run exited non-zero | `plan-proof` |
| `NEEDS_EVIDENCE` | The run passed and no pack exists | `read-evidence` |
| `NEEDS_REVIEW_PACKET` | The pack exists and no packet exists | `read-evidence` |
| `READY_FOR_HUMAN` | Every artifact is present and current | `read-evidence` |

## Step 1 — Prepare the change

**Instructions:** `capture-intent`

The agent reads the design document (or writes a short change description for ad-hoc work), extracts the requirements for this specific change, and writes `docs/changes/<change-id>-<slug>.md`.

The agent then records the change:

```bash
shipproof start <change-id> --intent docs/changes/<change-id>-<slug>.md --ceremony 1
shipproof status <change-id>
```

The `--ceremony` value sets how much proof the change needs. Pass `--ceremony 0`
for a trivial change. Level 0 needs no verification plan. Level 1 and above need
one.

To refresh a stale snapshot, add `--force`:

```bash
shipproof start <change-id> --intent docs/changes/<change-id>-<slug>.md --force
```

The `--force` option re-snapshots the source document and rewrites the record.
It keeps the recorded ceremony level, unless you also pass `--ceremony`.

## Step 2 — Plan verification

**Instructions:** `plan-proof`

The agent creates and populates the verification plan.

```bash
shipproof status <change-id>
```

At ceremony level 1 and above the verification plan is required. A change with
no plan stays at `NEEDS_PLAN`. Skip this step only at ceremony level 0.

## Step 3 — Implement and verify

**Instructions:** `plan-proof`

The agent reads the intent snapshot and verification plan, then makes the smallest coherent change that satisfies the approved scope. The agent then runs the repository verification contract and confirms the intent snapshot is intact:

```bash
shipproof prove <change-id>
shipproof prove <change-id> --gate-only
shipproof prove <change-id> --proofs-only
shipproof status <change-id>
```

`--gate-only` skips the attribution pass. `--proofs-only` skips the gate. Do
not use both flags together.

`shipproof status <change-id>` reports what each requirement proved at the
current revision.

## Unexplained change

ShipProof reports which changed code no approved proof ran. It never reports
which code answers to no requirement. Nothing can measure that.

The report has two resolutions. The line-level section needs a coverage command
in `.shipproof/config.yaml`. It names each changed line inside an instrumented
block with a zero count. Its provenance is `observed`. The file-level section is
always available. It names each changed file that no proof target names and no
coverage profile covers. Its provenance is `derived`.

The report always states how many changed lines it could not judge. A line
outside every instrumented block carries no claim.

The signal never fails a change. It is review material. Read it in
`review-packet.json` and in the change report.

Configure it like this:

```yaml
verification:
  coverage:
    command: go test -coverpkg=./... -coverprofile={{profile}} ./{{target}}/
    format: go
  unexplained_ignore:
    - "docs/**"
```

`{{profile}}` takes the output profile path. `{{target}}` takes the proof
target. An ignored path appears in the report with its reason. It counts toward
no total.

## Step 4 — Produce evidence

**Instructions:** `read-evidence`

The agent assembles the evidence pack from intent, implementation, and verification data:

```bash
shipproof pack <change-id>
```

ShipProof reads the base revision from the recorded agent run. Pass `--base <rev>` when no agent run recorded one. Without a base revision the pack carries no unexplained-change section, and the command says so on stderr.

## Step 5 — Code review and commit

**Instructions:** `read-evidence`

The agent reviews the implementation against the approved intent. After review, commit the implementation, change record, and evidence artifacts.

## Which skill to use at each step

| Step | Skill | CLI commands |
|---|---|---|
| Prepare | `capture-intent` | `start`, `status` |
| Plan verification | `plan-proof` | `status` |
| Implement | `plan-proof` | `prove`, `status` |
| Evidence | `read-evidence` | `pack` |
| Code review | `read-evidence` | none |

## Starting a new session

Start a new coding-agent session when:

- the current bounded change is complete;
- the conversation carries substantial implementation history that is already in files;
- a materially different work item begins.

The repository is the source of truth. Do not rely on chat history for durable decisions.
