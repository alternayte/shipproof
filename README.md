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

Four steps, in your own repository. Each one prints what to do next.

**1. Set up.**

```bash
cd your-project
shipproof init .
```

It creates `.shipproof/` and picks your test command by looking for a
`justfile`, a `Makefile`, a `package.json`, a `go.mod`, a `Cargo.toml`, or a
`pyproject.toml`. It tells you which one it chose. If it found none, put your
test command in `.shipproof/config.yaml` under `verification.command`.

**2. Point it at what you meant to build.**

```bash
shipproof start SP-1 --intent docs/my-feature.md
```

Any document works: an OpenSpec proposal, a Spec Kit specification, or a plain
Markdown list. ShipProof records its hash and proposes the requirements it can
read.

```
Requirements: 1 proposed, 0 adopted.
  SP-1-R1  MUST retry a failed charge.
Read each line. ShipProof matched a pattern; it judged nothing.
Confirm them with:
  shipproof start SP-1 --confirm-requirements
```

A proposal is never a fact. Read the lines, then confirm them. To accept only
some, edit `.shipproof/changes/SP-1/requirements-proposal.json` first and
delete the rest.

**3. Say how each requirement gets proven.**

Open `.shipproof/changes/SP-1/verification.json` and give each requirement a
command that exits 0 when it holds. Mark a requirement human when no command
can judge it.

Not sure what is left? Ask.

```bash
shipproof status SP-1
```

```
needs a proof   SP-1-R1
                add a command that exits 0 to .shipproof/changes/SP-1/verification.json
```

It also reports a `broken proof`: a command whose program is not installed.
That is a broken proof, not a failed requirement, and ShipProof says so before
it records anything.

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

`pack` runs the proofs for you, so `start` and `pack` are enough.

## When the document changes

Requirements move. Re-run `start` with `--force`:

```bash
shipproof start SP-1 --intent docs/my-feature.md --force
```

It merges. It adds what the document gained and keeps every requirement that
stands, with the confirmation you already gave.

```
Requirements: added 1 from the document: SP-1-R2
```

It **never deletes**. A requirement the document no longer states is reported
and left in place:

```
Requirements: the document no longer states SP-1-R2. Nothing was deleted.
              Remove it yourself if it is out of scope.
```

Without this, a stale requirement set would sit behind a fresh document hash,
and the pack would report on requirements nobody asked for any more.

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
shipproof start SP-002 --confirm-requirements
shipproof start SP-002 --intent docs/changes/SP-002-retries.md --force
```

`start` records an intent snapshot with its SHA-256 hash, and it creates the
verification plan for the agent to fill.

It adopts the requirement set when the document uses ShipProof's own heading
format. For any other document it reads one documented pattern, a list item
that states an obligation with MUST or SHALL, and writes a **proposal**. A
pattern match is never a fact, so the proposal waits for
`--confirm-requirements`. Edit
`.shipproof/changes/<id>/requirements-proposal.json` first to accept a subset.

`--force` re-snapshots the document and merges the requirement set with it. It
adds what the document gained, keeps every requirement that stands with its
confirmation, and never deletes. A requirement the document no longer states is
reported and left in place.

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

`status` opens with the verdict block, then reports the state, the exact next
command, the instruction file that covers it, and the requirement coverage.

It also names what stands between the change and a verdict: every requirement
with no proof, and every planned proof whose program is not installed. A
command that cannot run is a broken proof, not a failed requirement, and
`status` says so before `prove` records anything.

ShipProof stores no cursor, so the answer stays correct when an agent acts out
of band.

### Agent instructions

ShipProof ships three instruction files. `init` installs them for five harness
targets: Claude Code, Cursor, Codex, OpenCode, and a plain `AGENTS.md`.

| Skill | What it covers |
|---|---|
| `capture-intent` | How to record the intent document and its requirements. |
| `plan-proof` | How to map each requirement to a runnable proof in `verification.json`, with a worked example your agent can copy. |
| `read-evidence` | How to read a pack, and how to report the verdict without inflating it. |

Each installs as `<name>/SKILL.md` with frontmatter, so your coding agent loads
it when the task matches. If you use a coding agent, `plan-proof` is the one
that fills in `verification.json` for you.

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
