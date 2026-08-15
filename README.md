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

### Document review

```bash
shipproof doc status docs/prd/retries.md
shipproof doc review docs/prd/retries.md
shipproof doc review design.md --kind sdd --json
```

Deterministic checks catch structural gaps (missing problem statement, absent scope), unresolved placeholders, and contextless quality attributes. Semantic review is performed by the portable Agent Skills inside the coding harness.

The CLI does not invoke an LLM. A clean deterministic result does not guarantee semantic completeness.

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
shipproof change start SP-002 --source docs/prd/retries.md
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
```

`shipproof verify` runs the configured repository verification command. With a change ID it behaves like `verification run` and writes a structured run result. Without one, logs go to `.shipproof/runs/adhoc/` and the command's exit code is returned.

Verification plans map requirements and invariants to proof before implementation. Each plan item requires at least one proof with a type and target.

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

ShipProof ships with 15 portable Agent Skills. Install them into the harness discovery path for Claude Code (`.claude/skills/`), Cursor or Codex (`.agents/skills/`). Modified skill files are not overwritten unless `--force` is explicit.

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
| `implement-change` | Implement one approved change against its verification plan. |
| `verify-change` | Verify a change with deterministic checks. |
| `review-change` | Review an implemented change for correctness and agent failure patterns. |
| `prepare-human-review` | Prepare a focused human-review packet. |
| `produce-evidence` | Produce a versioned evidence pack from recorded facts. |
| `benchmark-run` | Run one benchmark task from a fixed commit and record the result without exposing hidden evaluation material. |
| `triage-change` | Assess a feature, fix, or issue and recommend the smallest useful ceremony level. |
| `prepare-change` | Prepare the next ShipProof change from approved intent. |

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
