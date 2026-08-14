# SP-008 — Agent telemetry collection

## Problem

ShipProof captures intent, git metadata, and verification results in the evidence pack. It does not capture agent session metadata: how long the session took, which model ran, how many tokens it used, or what it cost. Without this data, the evidence pack cannot answer questions about agent efficiency or cost.

## Desired outcome

A developer can collect telemetry from a completed agent session and include it in the evidence pack for a change.

## Scope

Implement post-hoc telemetry collection from local harness session data.

Do not wrap or hook agent invocation.

Do not add a real-time telemetry pipeline.

## Requirements

### SP-008-R1 — Telemetry collect command

`shipproof telemetry collect <change-id> --adapter <adapter>` reads the most recent session data from the specified harness and writes `.shipproof/runs/<change-id>/agent-run.json`.

### SP-008-R2 — Telemetry contract

The agent run record contains these fields when the harness provides them:

- provider
- agent version
- model
- start and end timestamps
- session ID
- cost
- token usage (input and output)
- tool call count
- exit status
- raw log reference

### SP-008-R3 — Missing fields stay unknown

Fields that the harness does not expose remain absent from the record. ShipProof does not estimate or infer missing values.

### SP-008-R4 — Claude Code adapter

The Claude Code adapter reads session data from the local Claude Code session storage and extracts the available telemetry fields.

### SP-008-R5 — OpenCode adapter

The OpenCode adapter reads session data from the local OpenCode session storage and extracts the available telemetry fields.

### SP-008-R6 — Evidence pack integration

`shipproof evidence pack` includes agent run metadata when an agent run record exists for the change.

## Acceptance

Automated tests must cover record creation, missing field handling, and evidence pack integration.

`go test -race ./...`, `go vet ./...`, formatting checks, and build must pass.

## Non-goals

- Cursor adapter (add later when session storage format is documented).
- Codex adapter (add later when session storage format is documented).
- Real-time streaming of agent events.
- Cost estimation when the harness does not record cost.
- Wrapping or hooking agent invocation.

## Risk

Agent harness session storage formats are not stable APIs. Adapters can break when the harness updates. Keep each adapter small and isolated so breakage is contained.
