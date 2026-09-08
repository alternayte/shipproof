# ShipProof

Evidence for AI-assisted software delivery.

ShipProof is a CLI tool that helps teams produce verifiable evidence of what was built, why, and whether it works. It provides five commands, portable Agent Skills, and a versioned evidence contract that keeps AI-assisted work auditable.

## Quick start

```bash
go install github.com/alternayte/shipproof/cmd/shipproof@latest
shipproof init .
```

`shipproof init` creates a `.shipproof/` directory with templates and a repository verification command, and it installs the ShipProof instructions into every harness path. It never overwrites existing files.

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

The whole product is five commands and two support commands.

```bash
shipproof init [directory]
shipproof start <change-id> --intent <path>
shipproof prove [change-id]
shipproof pack [change-id]
shipproof status [change-id]

shipproof runner <list|doctor>
shipproof config <get|set> <key> [value]
```

### `init`

`init` prepares the repository and installs the ShipProof instructions for
Claude Code (`.claude/skills/`), OpenCode (`.opencode/skills/`), and the
portable `.agents/skills/` path that Cursor and Codex read. It never
overwrites a modified file.

### `start`

```bash
shipproof start SP-002 --intent docs/changes/SP-002-retries.md
shipproof start SP-002 --intent docs/changes/SP-002-retries.md --force
```

`start` records an immutable intent snapshot with its SHA-256 hash. It adopts
the requirement set when the document names requirement identifiers, and it
creates the verification plan for the agent to fill. `--force` re-snapshots a
stale source document.

### `prove`

```bash
shipproof prove SP-002
shipproof prove SP-002 --gate-only
shipproof prove SP-002 --proofs-only
```

`prove` validates the verification plan, runs the repository gate, and runs
each proof on its own. The gate decides whether the repository passes. The
attribution pass records one result per proof. A green attribution never masks
a red gate. With no change identifier, `prove` resolves the single open change.

### `pack`

```bash
shipproof pack SP-002
shipproof pack SP-002 --base main
shipproof pack SP-002 --adapter claude
```

`pack` collects the agent telemetry when you name an adapter, assembles the
evidence pack, and writes the HTML change report beside it. Every metric
carries a provenance label.

### `status`

```bash
shipproof status SP-002
shipproof status SP-002 --json
```

`status` derives the current phase from the artifacts on disk. It reports the
phase, the blocker, the exact next command, the skill that handles it, and the
requirement coverage. ShipProof stores no cursor, so the answer stays correct
when an agent acts out of band.

### Agent instructions

ShipProof ships three instruction files. `init` installs them for five harness
targets: Claude Code, Cursor, Codex, OpenCode, and a plain `AGENTS.md`.

| File | What it covers |
|---|---|
| `capture-intent.md` | How to record the intent document and its requirements. |
| `plan-proof.md` | How to map each requirement to a runnable proof, and how to mark a proof as human. |
| `read-evidence.md` | How to read a pack, and how to report the verdict without inflating it. |

One rule sits in all three files. An agent must never write a result that a
tool did not produce.

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
  changes/    Change records with intent snapshots and verification plans.
  evidence/   Evidence packs with provenance labels.
  templates/  PRD and SDD reference templates.
  skills/     Canonical skill copies installed at harness time.
```

## License

MIT
