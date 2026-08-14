# SP-002 — Verification runner

## Problem

ShipProof can start a change and snapshot its intent, but it cannot yet run the
repository-owned verification command and record the result.

Without a verification runner, each change has no automated evidence of whether
it passes the project checks.

## Desired outcome

A developer runs `shipproof verification run <change-id>`. ShipProof executes
the command from `.shipproof/config.yaml`, captures exit status, duration,
stdout, and stderr, and writes structured run results.

## Scope

Implement the process runner and result recording. Do not parse JUnit, SARIF,
or Git evidence in this change. Do not assemble evidence packs.

## Requirements

### SP-002-R1 — Read verification command from config

Read `verification.command` from `.shipproof/config.yaml`. Return a clear error
when the key is missing or the value is empty.

### SP-002-R2 — Execute the command

Run the command through the default shell. Accept space-separated arguments.

### SP-002-R3 — Record exit code

Capture the process exit code. Treat exit code 0 as pass.

### SP-002-R4 — Record wall-clock duration

Measure elapsed time from command start to command end.

### SP-002-R5 — Capture stdout and stderr

Write stdout to `.shipproof/runs/<change-id>/stdout.log` and stderr to
`.shipproof/runs/<change-id>/stderr.log`.

### SP-002-R6 — Associate run with a change

Accept `<change-id>` as a required argument. Verify the change record exists
before running.

### SP-002-R7 — Write structured run results

Write `.shipproof/runs/<change-id>/run.json` with schema version, change ID,
exit code, duration, output file references, and timestamp.

### SP-002-R8 — CLI surface

`shipproof verification run <change-id>` invokes the runner.

## Acceptance

Automated tests must cover config loading, command execution with success and
failure exit codes, missing command error, missing change error, and output
file creation.

`go test -race ./...`, `go vet ./...`, formatting checks, and build must pass.

## Non-goals

- Parsing structured test output.
- Collecting Git evidence.
- Assembling evidence packs.
- Agent telemetry.
- Running verification automatically on change start.
- Running verification without a change.

## Risk

The command runs through the default shell. The working directory is the
repository root. This is the standard CI behavior.
