# The hook contract

ShipProof runs at a natural point in your workflow: the end of an agent
session, or the moment before a commit. A hook does that for you.

**The hook is optional. The command is the contract.**

A hook runs a ShipProof command. It never calls a ShipProof library, and no Go
package is part of the contract. Every example below runs the binary, and a
test in this repository runs every command that this document names.

## What this document guarantees

ShipProof verifies the command, not the configuration format around it.

The command is stable, and a test proves it. The file format that each harness
reads belongs to that harness, and it changes with the harness version. Check
the block you copy against your harness documentation before you rely on it.

## The two commands

Section 8.1 of the design document names two commands for a hook.

| Command | When to run it |
|---|---|
| `shipproof prove` | After the code changes. It runs the gate and each proof. |
| `shipproof pack` | At the end. It runs `prove` when no fresh result exists, then writes the pack and the report. |

`pack` alone is enough for most hooks, because it runs `prove` for you.

## Exit codes

A hook must read the exit code and decide what to do with it.

| Exit code | Meaning | What a hook must do |
|---|---|---|
| `0` | The command ran and recorded what it saw. | Read the verdict. |
| `1` | The command could not finish. A file was unreadable, or the state was inconsistent. | Show the output and stop. |
| `2` | The command was used wrongly, or no open change exists, or several do. | Name the change in the hook, or start one. |

**An exit code of `0` does not mean `PROVEN`, and a failed proof does not
change it.** `prove` exits `0` when it ran each proof and recorded the result.
Running and recording is its whole job. A proof that failed is a recorded
result, not a broken command.

The verdict is the answer. Read it with `shipproof status`, or read
`verdict.verdict` from `shipproof status --json`. A hook that blocks on a
failure must test the verdict, never the exit code of `prove`.

## Claude Code

Claude Code reads hooks from `.claude/settings.json`. Run ShipProof when the
session stops.

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "shipproof pack"
          }
        ]
      }
    ]
  }
}
```

```
#> shipproof pack
```

## Cursor

Cursor reads hooks from `.cursor/hooks.json`. Run ShipProof after the agent
stops.

```json
{
  "version": 1,
  "hooks": {
    "stop": [
      { "command": "shipproof pack" }
    ]
  }
}
```

```
#> shipproof pack
```

## Codex

Codex runs a notification program from `~/.codex/config.toml`. Point it at a
script that runs ShipProof.

```toml
notify = ["/usr/local/bin/shipproof-notify"]
```

```bash
#!/usr/bin/env bash
# /usr/local/bin/shipproof-notify
set -euo pipefail
cd "${CODEX_WORKSPACE:-$PWD}"
#> shipproof pack
```

## OpenCode

OpenCode runs a plugin at the end of a session. The plugin runs the command; it
imports nothing from ShipProof.

```javascript
export const shipproof = async ({ $ }) => ({
  event: async ({ event }) => {
    if (event.type === "session.idle") {
      await $`shipproof pack`
    }
  },
})
```

```
#> shipproof pack
```

## A plain AGENTS.md project

A git hook works in every repository, whatever agent wrote the code. This one
runs the proofs and stops the commit when the verdict reads `FAILED`. The
repository ships it as `scripts/pre-commit`.

```bash
#!/usr/bin/env bash
# .git/hooks/pre-commit
set -euo pipefail
#> shipproof prove
if shipproof status --json | grep -q '"verdict": "FAILED"'; then
  shipproof status >&2
  exit 1
fi
```

The script runs `prove`, then reads the verdict and stops the commit when it
reads `FAILED`. It does not test the exit code of `prove`, because `prove`
exits `0` after it records a failure.

One caveat about a commit hook. The commit creates a new revision, so the run
that the hook recorded describes the revision before it. ShipProof then reports
the result as not current, which is honest and correct. A commit hook is
therefore a fast local signal, and it is not the authoritative gate.
Continuous integration is, because it runs against the revision it judges. See
Section 9 of the design document.

Install it:

```bash
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## Writing your own hook

1. Run `shipproof pack` at the point you care about.
2. Read the exit code from the table above.
3. Read the verdict with `shipproof status` when you want the outcome in
   words.
4. Never parse the human output. Use `shipproof status --json`, or read
   `.shipproof/changes/<id>/evidence-pack.json`.

The pack is the stable interface. The human output is for a person.
