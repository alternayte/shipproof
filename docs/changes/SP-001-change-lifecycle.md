# SP-001 — Change lifecycle and intent snapshot

## Problem

ShipProof can review intent and keep shaping state, but it cannot yet create a durable execution unit for one implementation change.

Without that unit, later evidence cannot prove which intent revision the agent implemented.

## Desired outcome

A developer can start `SP-001` from an approved source document or issue description.

ShipProof creates a local change record and an immutable intent snapshot with a content hash.

Later verification and evidence can reference that snapshot.

## Scope

Implement local repository behavior only.

Do not add Linear, GitHub, remote storage, or an agent SDK in this change.

Use repository files as the source of truth.

## Requirements

### SP-001-R1 — Start a change

`shipproof change start <change-id> --source <path>` creates `.shipproof/changes/<change-id>/change.json`.

### SP-001-R2 — Snapshot intent

ShipProof copies the source content into the change directory without modifying the source.

### SP-001-R3 — Record provenance

The change record contains the source path, SHA-256 content hash, capture time, and snapshot path.

### SP-001-R4 — Protect history

Starting an existing change fails without overwriting its record or snapshot.

### SP-001-R5 — Inspect state

`shipproof change status <change-id>` shows the source, snapshot hash, and verification-plan presence.

### SP-001-R6 — Validate state

`shipproof change check <change-id>` validates the change record and confirms that the current snapshot content matches the recorded hash.

## Acceptance

Automated tests must cover creation, hashing, duplicate protection, status loading, and tamper detection.

`go test -race ./...`, `go vet ./...`, formatting checks, and build must pass.

## Non-goals

- Linear synchronization.
- Pull-request creation.
- Agent invocation.
- Evidence-pack generation.
- Source-document change detection after capture.
- Multiple intent snapshots for one change.

## Risk

The file format becomes a long-lived contract. Keep it minimal and version it from the first implementation.
