# ShipProof

Evidence for AI-assisted software delivery.

ShipProof is a CLI tool that helps teams produce verifiable evidence of what was built, why, and whether it works. It provides portable Agent Skills, deterministic document checks, and a versioned evidence contract that keeps AI-assisted work auditable.

## Quick start

```bash
go install github.com/shipproof/shipproof/cmd/shipproof@latest
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

### Verification plans

```bash
shipproof verification init SP-002
shipproof verification check SP-002
```

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
```

ShipProof ships with 12 portable Agent Skills. Install them into the harness discovery path for Claude Code (`.claude/skills/`), Cursor or Codex (`.agents/skills/`). Modified skill files are not overwritten unless `--force` is explicit.

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
