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
default config, and it installs the ShipProof instructions into every harness
path. It never overwrites existing files.

Keep a small `AGENTS.md` with repository invariants. Skills define task
workflows; `AGENTS.md` keeps static context small.

Edit `.shipproof/config.yaml`:

```yaml
verification:
  command: just verify        # your repository's verification entry point
evidence:
  capture: metadata           # metadata | redacted | full
```

`just verify` is the recommended convention. Any shell command works.

## The workflow in one pass

```bash
shipproof start WEB-142 --intent docs/changes/web-142.md
shipproof status WEB-142
shipproof prove WEB-142
shipproof pack WEB-142 --base main --adapter claude
```

The `plan` and `implement` steps run inside your coding agent through the
installed skills. The CLI handles deterministic state, evidence, and reports.
`shipproof status` names the next command at every point.

## What ShipProof does not do

ShipProof does not replace your issue tracker, CI, or agent. It records
intent, runs your verification command, collects evidence, and prepares
focused review material. It never changes a failing deterministic check
into a pass, and it never estimates cost as observed fact.
