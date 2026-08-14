# SP-007 — Linear adapter

## Problem

ShipProof can capture intent, run verification, collect Git evidence, and
assemble evidence packs. It cannot yet read issue-tracker data or sync approved
change plans to an external tracker. The SDD designates Linear as the first
integration target. Without an adapter, decomposed plans from the
`decompose-plan` skill stay in local files with no connection to the
organization's issue tracker.

## Desired outcome

`shipproof linear issue <identifier>` reads a Linear issue and prints its title,
description, state, and linked documents. `shipproof linear project <name>` reads
a Linear project and prints its issues and metadata. `shipproof linear sync
<plan-file>` reads a decomposed plan, asks for human approval, and creates issues
with parent-child relationships in Linear.

## Scope

Implement the Linear adapter in a new package `internal/linear/`. Wire it to the
CLI as `shipproof linear`. Keep Linear GraphQL types inside the adapter. Do not
add Linear concepts to the core schema or evidence packages. Use only standard
library HTTP. Authenticate with a `LINEAR_API_KEY` environment variable or a
`.shipproof/config.yaml` field.

## Requirements

### SP-007-R1 — Read a Linear issue

Accept an issue identifier in the format `TEAM-123`. Call the Linear GraphQL API
to fetch the issue's title, description, state name, assignee name, cycle number,
labels, and linked documents. Print the result as formatted JSON to stdout.
Return a clear error when the identifier is invalid, the issue does not exist, or
the API call fails.

### SP-007-R2 — Read a Linear project

Accept a project name or slug. Call the Linear GraphQL API to fetch the
project's name, description, state, lead name, and a list of its issues with
titles and states. Print the result as formatted JSON to stdout. Return a clear
error when the project does not exist.

### SP-007-R3 — Read linked documents

When an issue has attached Linear documents, include each document's title,
content excerpt, and URL in the issue output. When no documents exist, omit the
field rather than printing null.

### SP-007-R4 — GraphQL client with token auth

Implement a minimal GraphQL client. Send POST requests to
`https://api.linear.app/graphql` with a JSON body containing `query` and
`variables`. Set the `Authorization` header to the API key from the
`LINEAR_API_KEY` environment variable. Support a config file fallback from
`.shipproof/config.yaml` field `linear.api_key`. Return a clear error when no
API key is available.

### SP-007-R5 — Linear types are adapter-private

Define Linear-specific types (issue, project, document, state, user) inside
`internal/linear/`. Do not expose these types to `internal/schema/`,
`internal/evidence/`, or any other core package. The adapter is an outbound
integration layer.

### SP-007-R6 — Sync a decomposed plan

Accept a path to a decomposed plan file. The plan file is a JSON array of issue
objects. Each object contains `title`, `description`, and optional
`children` (array of sub-issue objects with `title` and `description`). Ask the
user for confirmation before creating any issues. On approval, create a Linear
project for the workstream, then create issues with parent-child relationships. 
Print each created issue's identifier and URL.

### SP-007-R7 — Approval prompt for sync

Before creating or changing any Linear work items, print a summary of what will
be created and ask for explicit confirmation on stderr. Read `y` or `yes` from
stdin. Any other input cancels the operation. Do not create work items without
this approval.

### SP-007-R8 — Handle API errors gracefully

For all API operations, handle network errors, authentication failures (HTTP
401), rate limits (HTTP 429), and unexpected GraphQL errors. Return clear,
actionable messages rather than raw HTTP details.

### SP-007-R9 — CLI surface

Add `shipproof linear` to the CLI with subcommands:
- `shipproof linear issue <identifier>` — read an issue
- `shipproof linear project <name>` — read a project
- `shipproof linear sync <plan-file>` — sync a decomposed plan

## Acceptance

`go test -race ./...`, `go vet ./...`, formatting checks, and `just verify`
must pass.

Unit tests must cover:
- GraphQL request construction and response parsing with mock HTTP servers.
- Issue read with valid, missing, and error responses.
- Project read with valid and missing responses.
- API key resolution from environment variable and config file.
- Sync plan loading from a valid and missing plan file.
- Sync approval prompt acceptance and rejection.
- CLI argument validation for all subcommands.
- Linear types do not leak into schema or evidence packages (package dependency
  check).

## Non-goals

- Posting evidence summaries or links to Linear issues.
- Reading Linear comments or workflow states.
- OAuth authentication.
- Webhooks or real-time sync.
- Bidirectional sync (Linear changes back to ShipProof).
- Updating existing issues.
- Importing linked intent from Linear documents into ShipProof plans.
- Detecting the Linear team identifier automatically (the caller must supply it).

## Risk

SP-007 depends on the Linear GraphQL API, which can change. Unit tests use mock
HTTP servers to avoid network dependencies. The adapter must tolerate schema
additions in the API response. The sync operation creates real issues and must
not run without explicit human approval.
