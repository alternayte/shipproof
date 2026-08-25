# ShipProof

Evidence for AI-assisted software delivery.

ShipProof is a CLI tool that helps teams produce verifiable evidence of what was built, why, and whether it works. It provides portable Agent Skills, deterministic document checks, and a versioned evidence contract that keeps AI-assisted work auditable.

## Quick start

```bash
go install github.com/alternayte/shipproof/cmd/shipproof@latest
shipproof init .
```

`shipproof init` creates a `.shipproof/` directory with templates, a glossary, and a repository verification command. It never overwrites existing files.

## Concepts

### Readiness states

Every document has a finite readiness state:

| State | Meaning |
|---|---|
| `SHAPING` | Material information is still being gathered. |
| `BLOCKED` | A missing decision or contradiction prevents responsible progression. |
| `READY_WITH_ASSUMPTIONS` | No blocker remains but accepted assumptions or known risks exist. |
| `READY` | No blocker remains for the next stage. |

Blockers and unresolved decisions prevent readiness. Suggestions and nits do not.

### Finding classes

ShipProof classifies every finding so teams know what matters:

`BLOCKER`, `DECISION`, `ASSUMPTION`, `RISK`, `SUGGESTION`, `NIT`.

### Evidence provenance

Every piece of evidence carries a provenance label: `observed`, `derived`, `inferred`, or `human`. These labels prevent generated estimates from being confused with measured results.

## Commands

### Delivery phase

```bash
shipproof next
shipproof next SP-002
shipproof next SP-002 --json
```

`next` derives the current delivery phase from the artifacts on disk. It reports
the phase, the blocker, the exact next command, and the skill that handles it.
ShipProof stores no cursor, so the answer stays correct when an agent acts out
of band.

With no change identifier, `next` resolves the single change that is not
`READY_FOR_HUMAN`.

### Document review

```bash
shipproof doc status docs/prd/retries.md
shipproof doc review docs/prd/retries.md
shipproof doc review design.md --kind sdd --json
shipproof doc adopt SP-002 --source docs/changes/SP-002-retries.md
shipproof doc adopt SP-002 --source design.md --confirm
```

Deterministic checks catch structural gaps (missing problem statement, absent scope), unresolved placeholders, and contextless quality attributes. Semantic review is performed by the portable Agent Skills inside the coding harness.

The CLI does not invoke an LLM. A clean deterministic result does not guarantee semantic completeness.

`doc adopt` extracts the requirement set from a source document into `.shipproof/changes/<change-id>/requirements.json`. A document in the `docs/changes/` format adopts with `observed` provenance and no human step. Any other document prints a proposal and needs `--confirm`. With `--confirm`, it writes with `human` provenance. `doc adopt` refuses to overwrite an existing requirement set. Pass `--force` to replace it.

### Shaping sessions

```bash
shipproof shape prd "Webhook retries" --id webhook-retries
shipproof shape sdd "Secret rotation" --id secret-rotation --source design.md
shipproof shape status webhook-retries
shipproof shape check webhook-retries
```

Shaping sessions persist under `.shipproof/shaping/` as compact JSON decision ledgers. Each session tracks decisions, assumptions, risks, unknowns, and readiness state. State transitions are validated for consistency.

### Change management

```bash
shipproof change start SP-002 --source docs/prd/retries.md --ceremony 1
shipproof change status SP-002
shipproof change check SP-002
```

Each change captures an immutable intent snapshot with SHA-256 provenance. Change records live under `.shipproof/changes/<change-id>/`.

`change status` and `change check` also report intent staleness. When the source document changes after the snapshot, the intent is stale and the change needs re-verification. Evidence packs carry an `intent:staleness` check.

### Plans

```bash
shipproof plan create docs/design/delivery-plan.md
shipproof plan review
shipproof plan sync --linear [issues.json]
```

`plan create` snapshots a design document as a plan record under `.shipproof/plans/`. `plan review` validates every plan record, its snapshot hash, and its source staleness. `plan sync --linear` creates the Linear project and issues from a decomposed issue list after human approval. Without an explicit file, it uses the single `issues.json` under `.shipproof/plans/`.

### Verification plans

```bash
shipproof verify
shipproof verify SP-002
shipproof verification init SP-002
shipproof verification check SP-002
shipproof verification run SP-002 --gate-only
shipproof verification run SP-002 --proofs-only
shipproof coverage SP-002
shipproof coverage SP-002 --json
```

`shipproof verify` runs the configured repository verification command. With a change ID it behaves like `verification run` and writes a structured run result. Without one, logs go to `.shipproof/runs/adhoc/` and the command's exit code is returned.

Verification plans map requirements and invariants to proof before implementation. Each plan item requires at least one proof with a type and target. A proof carries either a `command`, or `human: true` with a `rationale`. `verification check` rejects a proof that carries neither.

`verification check` also compares the requirement set against the plan when a requirement sidecar exists. A requirement with no plan entry blocks. A plan entry with no requirement blocks. Invariants take no part in the tie check.

`verification run` performs two jobs. The gate runs the repository verification command and decides whether the repository passes. The attribution pass runs each proof on its own and records one result per proof in `.shipproof/runs/<change-id>/proofs.json`. A green attribution never masks a red gate. Use `--gate-only` to skip the attribution pass, and `--proofs-only` to skip the gate. Both flags together is an error.

`shipproof coverage <change-id>` reports what each requirement proved at the current revision. It derives the matrix on demand and writes nothing.

### Reports

```bash
shipproof report change SP-002
shipproof report change SP-002 --output report.html
shipproof report pr-summary SP-002
shipproof report project my-project
```

Each report includes provenance badges on every metric. HTML change reports show intent, verification, implementation, and agent-run metadata. Markdown PR summaries answer the five SDD review questions. Project aggregate reports derive pass rates, agent usage, and cost across all changes.

### Skills

```bash
shipproof harness install claude
shipproof harness install cursor
shipproof harness install codex
shipproof skill check
shipproof skill eval list
shipproof skill eval record prd-ready-stop --condition without --file result.json
shipproof skill eval results --regression
```

ShipProof ships with 14 portable Agent Skills. Install them into the harness discovery path for Claude Code (`.claude/skills/`), Cursor or Codex (`.agents/skills/`). Modified skill files are not overwritten unless `--force` is explicit.

Skill eval runs are recorded under `benchmarks/skill-evals/` with conditions `without`, `previous`, and `candidate`. `skill eval results --regression` flags candidate runs that fail a task the baseline passed, recall fewer blockers, or raise the false blocker rate.

Built-in skills cover the full delivery cycle:

| Skill | Purpose |
|---|---|
| `shape-prd` | Shape a PRD through a bounded interview. |
| `review-prd` | Review a PRD for material product-intent defects. |
| `shape-sdd` | Shape an SDD through a bounded technical interview. |
| `review-sdd` | Review an SDD for correctness and design gaps. |
| `record-decision` | Create an architecture decision record. |
| `decompose-plan` | Decompose intent into independently verifiable changes. |
| `plan-verification` | Plan how to prove a change before implementation. |
| `implement-change` | Implement one approved change against its verification plan, then verify it. |
| `review-change` | Review an implemented change for correctness and agent failure patterns. |
| `prepare-human-review` | Prepare a focused human-review packet. |
| `produce-evidence` | Produce a versioned evidence pack from recorded facts. |
| `benchmark-run` | Run one benchmark task from a fixed commit and record the result without exposing hidden evaluation material. |
| `triage-change` | Assess a feature, fix, or issue and recommend the smallest useful ceremony level. |
| `prepare-change` | Prepare the next ShipProof change from approved intent. |

### Agent execution

ShipProof can run a bounded change through your existing coding agent. It stays
independent of any one harness or provider.

```bash
shipproof runner list
shipproof runner doctor
shipproof config set agent.runner codex --global
shipproof config set agent.runner opencode --local
shipproof config get agent.runner
shipproof run SP-002
shipproof run SP-002 --runner claude
```

Runner resolution follows this precedence, highest first: the `--runner` flag,
the `SHIPPROOF_RUNNER` variable, `.shipproof/config.yaml`,
`~/.config/shipproof/config.yaml`, then auto-selection when exactly one usable
runner exists. ShipProof never chooses a vendor silently.

```yaml
agent:
  runner: codex
  review_runner: claude
  repair:
    max_attempts: 2
  runners:
    codex:
      model: ""
    claude: {}
    opencode:
      base_url: "http://127.0.0.1:4096"
```

v0 ships three adapters: Codex and Claude Code over a subprocess, and OpenCode
over its server transport.

ShipProof does not own model-provider authentication. Authenticate with the
coding agent itself. ShipProof never stores an API key, never copies a
credential file, and refuses a configuration key that looks like a secret.

`shipproof run` returns one of three outcomes:

- `PASS`: deterministic verification passed and the adversarial review ran.
- `NEEDS_REVIEW`: verification failed after the bounded repair attempts.
- `BLOCKED`: the runner is unusable, or it cannot enforce the role policy.

A runner claim is never evidence. ShipProof records the base and result Git
revisions, then runs its own verification. Adversarial reviewer findings enter
the evidence pack as agent-inferred. They are never labeled as observed.

## Evidence capture levels

```yaml
evidence:
  capture: metadata   # metadata | redacted | full
```

- `metadata`: store timing, provider, model, status, and evidence references. Do not store prompts or transcripts.
- `redacted`: also copy the raw session transcript under `.shipproof/runs/<change-id>/agent-raw/` with recognized secret shapes masked.
- `full`: copy the transcript unchanged where provider terms allow it.

The public reference application should use `full` where practical. Client repositories default to `metadata`.

## Design constraints

- Repository files are the source of truth.
- Core contracts do not depend on any issue tracker or agent vendor.
- Observed, derived, inferred, and human-supplied evidence stay distinct.
- A failing deterministic check is always reported as a failure.
- A document is complete enough when it supports the next decision.
- The language profile is STE-assisted. It does not claim ASD-STE100 certification.

## Build and verify

```bash
go build -o bin/shipproof ./cmd/shipproof
```

Run the full verification pipeline:

```bash
just verify
```

If `just` is not installed:

```bash
test -z "$(gofmt -l ./cmd ./internal ./skills)"
go vet ./...
go test -race ./...
go build ./cmd/shipproof
```

## Repository layout

```text
.shipproof/
  shaping/    Session state for PRD, SDD, and issue shaping.
  changes/    Change records with intent snapshots and verification plans.
  evidence/   Evidence packs with provenance labels.
  decisions/  Architecture decision records.
  templates/  PRD and SDD reference templates.
  skills/     Canonical skill copies installed at harness time.
```

## License

MIT
