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

`shipproof init` creates the `.shipproof/` directory with templates and a
default config. It never overwrites existing files.

Edit `.shipproof/config.yaml`:

```yaml
verification:
  command: just verify        # your repository's verification entry point
evidence:
  capture: metadata           # metadata | redacted | full
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

## The workflow in one pass

```bash
shipproof change start WEB-142 --source docs/changes/web-142.md
shipproof verification init WEB-142
shipproof verify WEB-142
shipproof telemetry collect WEB-142 --adapter claude
shipproof evidence pack WEB-142 --base main
shipproof review prepare WEB-142
shipproof report change WEB-142 --output report.html
```

The `plan` and `implement` steps run inside your coding agent through the
installed skills. The CLI handles deterministic state, evidence, and reports.

## What ShipProof does not do

ShipProof does not replace your issue tracker, CI, or agent. It records
intent, runs your verification command, collects evidence, and prepares
focused review material. It never changes a failing deterministic check
into a pass, and it never estimates cost as observed fact.
