# Changelog

ShipProof follows semantic versioning. Schema versions inside evidence files
stay at 0.1 during the v0.2 line.

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
