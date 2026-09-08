# ShipProof

Evidence for AI-assisted software delivery.

ShipProof reads the intent document your spec tool produced, watches what your
repository and your tools report, and publishes one evidence pack with an
honest verdict. It names every changed line that no requirement explains.

It answers one question that a spec tool does not:

> Does the shipped diff match the intent, and which parts remain unproven?

## Install

Pick one. Neither needs a clone of this repository.

**Download the binary.** No Go needed.

```bash
# macOS (Apple silicon). Change darwin_arm64 for your platform.
curl -fsSL -o shipproof.tar.gz \
  https://github.com/alternayte/shipproof/releases/latest/download/shipproof_darwin_arm64.tar.gz
tar -xzf shipproof.tar.gz shipproof
sudo mv shipproof /usr/local/bin/
shipproof version
```

Four platforms are published: `darwin_arm64`, `darwin_amd64`, `linux_arm64`,
and `linux_amd64`. Every release also carries `checksums.txt`.

**Or build it with Go.**

```bash
go install github.com/alternayte/shipproof/cmd/shipproof@latest
```

## Quickstart

Five steps, in your own repository. Each one prints what to do next.

**1. Set up.**

```bash
cd your-project
shipproof init .
```

It creates `.shipproof/`, and it picks a test command by looking for a
`justfile`, a `Makefile`, a `package.json`, a `go.mod`, a `Cargo.toml`, or a
`pyproject.toml`. It tells you which one it chose. If it found none, open
`.shipproof/config.yaml` and put your test command under
`verification.command`.

**2. Point it at what you meant to build.**

```bash
shipproof start SP-1 --intent docs/my-feature.md
```

Any document works: an OpenSpec proposal, a Spec Kit specification, or a plain
Markdown list. ShipProof records its hash, and it proposes the requirements it
can read. Confirm them:

```bash
shipproof start SP-1 --confirm-requirements
```

**3. Say how each requirement gets proven.**

Open `.shipproof/changes/SP-1/verification.json` and give each requirement a
command that exits 0 when it holds. Mark a requirement human when no command
can judge it.

**4. Get the answer.**

```bash
shipproof pack SP-1
```

It runs your tests, runs each proof, writes the evidence pack, and renders an
HTML report. You get three lines:

```
VERDICT: NOT PROVEN
2 of 7 requirements have no proof. 14 changed lines match no requirement.
NEXT: run `shipproof prove SP-1` after you add a proof for R3 and R5.
```

**5. Check where you stand at any time.**

```bash
shipproof status SP-1
```

That is the whole loop. `shipproof pack` runs step 4's proofs for you, so
`start` and `pack` are enough.

## The verdict

Three verdicts exist. No score exists, because a percentage invites a debate.

| Verdict | Meaning |
|---|---|
| `PROVEN` | Every requirement has a passing observed proof or an accepted human proof. No unexplained change remains. |
| `NOT PROVEN` | The work is incomplete or unproven. The reason is stated. |
| `FAILED` | A proof ran and failed, or your test command failed. |

## The three grades

Every check in a pack carries one grade, so a reader can tell a measurement
from an assertion.

| Grade | Meaning |
|---|---|
| `observed` | A tool ran and a machine recorded the result. |
| `stated` | A person accepted it and signed for it. |
| `claimed` | Someone asserted it and no tool confirmed it. |

A `claimed` check never proves a requirement. The report says so on the page.

## Unknown stays unknown

A field that no tool reported is absent from the pack, and `empty_sections`
states why. A count that ShipProof could not measure reads "not known". It
never reads zero.

## In continuous integration

A local pack is not audit-grade. Add the action, and the build system signs
the pack and posts the verdict on the pull request.

```yaml
- uses: alternayte/shipproof/action@main
```

See [docs/hooks.md](docs/hooks.md) to run ShipProof from your agent, and
[docs/controls.md](docs/controls.md) to map each pack field onto the audit
question it answers.

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
revisions, then runs its own verification. A reviewer finding enters the
evidence pack as a `claimed` check. It is never graded `observed`.

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
- An observed result, a stated result, and a claimed result stay distinct.
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
