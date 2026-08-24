# Changelog

ShipProof follows semantic versioning. Schema versions inside evidence files
stay at 0.1 during the v0 line.

## v0.3.0 — 2026-08-24

Portable agent execution. This release closes the v0 definition of done.

- `AgentRunner` interface with `Probe` and `Run`, capability probing, a runner
  registry, and five-level runner resolution: command line, environment,
  repository configuration, user configuration, then auto-selection.
- Runner adapters for Codex and Claude Code over a subprocess transport, and
  for OpenCode over the server transport.
- `shipproof config get` and `shipproof config set` at local and global scope.
  ShipProof never stores a provider credential and refuses a credential key.
- `shipproof runner list` and `shipproof runner doctor` report installation,
  authentication, version, and capabilities without printing a credential.
- `shipproof run <change-id>` executes a bounded coding task, then decides
  PASS, NEEDS_REVIEW, or BLOCKED from deterministic verification. A runner
  claim is never evidence.
- Role policy with explicit degradation. An unenforceable policy returns
  BLOCKED instead of a prompt instruction presented as an enforcement
  boundary.
- Adversarial reviewer findings enter the evidence pack as agent-inferred.
- Bounded repair loop with a default of two attempts.
- Runner-neutral durable execution records. A fresh session can build an
  execution context without `session_ref`.
- The `shape-prd` skill now carries a stop rule, an optional-detail list, and
  a blocker test. Optional detail no longer blocks a ready document.

Known limits in v0.3:

- Telemetry adapters for Cursor and Codex are not implemented yet.
- The Pi runner is not implemented yet.
- The reference application and public benchmarks are not included yet.
- The `assess` and `benchmark` commands are planned for a later release.

## v0.2.0 — 2026-08-15

First public release.

- Shaping: bounded PRD and SDD interviews with readiness gates, decision
  ledgers, and independent document review.
- Deterministic document checks and an STE-assisted language linter.
- Fifteen portable Agent Skills covering the delivery cycle.
- Change lifecycle with SHA-256 intent snapshots and staleness marking.
- Verification runner, plans, JUnit and SARIF parsing, and Git evidence.
- Versioned evidence packs with observed, derived, inferred, and human
  provenance.
- Focused human-review packets.
- Linear adapter: read issues and projects, sync approved plans with human
  confirmation, post evidence summaries.
- Agent telemetry for Claude Code and OpenCode with metadata, redacted, and
  full capture levels.
- Reports: HTML change reports, Markdown PR summaries, and project aggregate
  reports with provenance badges.
- Skill eval recording with with-skill versus without-skill comparison and
  regression detection.
- Harness installers for Claude Code, Cursor, Codex, OpenCode, and the generic
  agents directory.

Known limits in v0.2:

- Telemetry adapters for Cursor and Codex are not implemented yet.
- The reference application and public benchmarks are not included yet.
- The `assess` command is planned for a later release.
- Portable agent execution is not implemented yet.
