# Adopting ShipProof in a repository

This guide covers the smallest path to start producing evidence in an
existing repository.

## Install

```bash
go install github.com/alternayte/shipproof/cmd/shipproof@latest
```

Or download a release binary from the GitHub releases page.

## Initialize

```bash
cd your-repository
shipproof init
```

`shipproof init` creates the `.shipproof/` directory with templates, a
glossary, and a default config. It never overwrites existing files.

Edit `.shipproof/config.yaml`:

```yaml
verification:
  command: just verify        # your repository's verification entry point
evidence:
  capture: metadata           # metadata | redacted | full
language:
  profile: ste-assisted
```

`just verify` is the recommended convention. Any shell command works.

## Install skills into your harness

```bash
shipproof harness install claude     # .claude/skills/
shipproof harness install opencode   # .opencode/skills/
shipproof harness install cursor     # .agents/skills/
shipproof harness install codex      # .agents/skills/
```

Keep a small `AGENTS.md` with repository invariants. Skills define task
workflows; `AGENTS.md` keeps static context small.

## Optional: Linear integration

Export a personal API token:

```bash
export LINEAR_API_KEY=<token>
export LINEAR_TEAM_ID=<team-id>
```

Read work from Linear:

```bash
shipproof linear issue ENG-142
shipproof linear project webhooks
```

ShipProof asks for confirmation before it creates or changes Linear work
items.

## The workflow in one pass

```bash
shipproof shape prd "Webhook retries" --id webhook-retries
shipproof shape sdd "Secret rotation" --id secret-rotation --source design.md
shipproof doc status docs/prd/webhook-retries.md
shipproof doc review docs/prd/webhook-retries.md

shipproof change start WEB-142 --source docs/changes/web-142.md
shipproof verification init WEB-142
shipproof verify WEB-142
shipproof telemetry collect WEB-142 --adapter claude
shipproof evidence pack WEB-142 --base main
shipproof review prepare WEB-142
shipproof report change WEB-142 --output report.html
```

The `shape`, `review`, `decompose`, and `implement` steps run inside your
coding agent through the installed skills. The CLI handles deterministic
state, evidence, and reports.

## What ShipProof does not do

ShipProof does not replace your issue tracker, CI, or agent. It records
intent, runs your verification command, collects evidence, and prepares
focused review material. It never changes a failing deterministic check
into a pass, and it never estimates cost as observed fact.
