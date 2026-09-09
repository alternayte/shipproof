# Changelog

ShipProof follows semantic versioning. The schema version inside an evidence
file moves on its own, and each artifact names the version it answers to.

## v0.5.1 — 2026-09-09

A second reader review read the rebuilt report. None of the seven questions
from the first review came up, so the wording of v0.5.0 is proven rather than
plausible. Two new questions came up, and this release answers both.

### Fixed

- **`start --force` left the requirement set stale.** It re-snapshotted the
  document and touched nothing else, so a change whose document gained a
  requirement carried a fresh intent hash against a set the document no longer
  matched. The pack then reported on the wrong requirements and said nothing.
  `--force` now merges: it adds what the document gained and keeps every
  requirement that stands, with the confirmation a person already gave. It
  matches on the sentence rather than the identifier, because a document that
  gains a line renumbers everything after it.

  It never deletes. A requirement the document no longer states is reported
  and left in place, and a person removes it deliberately or not at all.

- **A broken proof read as a failed requirement.** A proof whose program is not
  installed recorded a failure, which is a different fact. `shipproof status`
  now names it before `prove` runs.

- **The report named the intent document without saying what it was.** A reader
  met "read from `checkout.md`" and asked what that file was. It now reads "the
  requirements document `docs/checkout-retry.md`".

- **The report did not fit a phone.** It carried no viewport tag, so a phone
  rendered it at desktop width. A reader met the page on a phone during the
  review. Narrow screens now get their own rules, and a wide table scrolls on
  its own rather than pushing the page sideways.

### Added

- `shipproof status` names every requirement with no proof, and the file to put
  one in.

### Unchanged on purpose

The readers said the page is not too wordy. Every change above lives in the
commands, and nothing was added to the report. A test now fails when any
explanation on the page grows past one short answer.

## v0.5.0 — 2026-09-09

The report explains itself. Three readers who had never used ShipProof read a
change report, and every one stated the verdict correctly. Row U1 of the
definition of done is met, and Section 14 is complete.

Every one of those readers then asked what the rest of the page meant. This
release answers their seven questions.

### Changed

- **The report explains itself.** Three readers who had never used ShipProof
  read the change report and every one stated the verdict correctly. Every one
  then asked what the rest of the page meant. Each section now carries one
  sentence that says what it is, before it shows a number. The verdict block
  is unchanged: it passed the review.
- The requirement rows show the line of the intent document each requirement
  came from, and the page names that document.
- The page states where it was produced: a developer machine or a build
  system, and which revision it judged.
- The unexplained-change section says what would measure a count it does not
  know, rather than only reporting that it is unknown.
### Breaking

- `schema_version` moves from `0.3` to `0.4`. `requirements[].source_anchor`
  is new and optional, and it holds the line of the intent document that
  states the requirement. `schemas/v0.4/` is published. A pack written by an
  older version still answers to the schema of its own version, so a recorded
  pack stays valid.

### Known limits

- The seven answers are written and not yet proven. `SP-039` asks for a second
  review with three new readers who answer the seven questions without asking.
  Until that runs, the wording is plausible, not verified.
- The comprehension review of 2026-09-09 was recorded as a summary rather than
  a per-page tally. `SP-036` says so, because it changes what the record
  proves.

## v0.4.1 — 2026-09-09

A patch release. v0.4.0 could not verify a real signature, so every pack that
continuous integration signed failed its own check. Upgrade from v0.4.0.

### Fixed

- **The verifier rejected a real cosign signature.** `cosign sign-blob`
  base64-encodes its output by default, so the certificate arrives as base64
  around the PEM rather than as the PEM itself. `shipproof pack --verify` read
  only raw PEM and refused a true signature. It now reads raw PEM, base64 of a
  PEM, and base64 of the raw DER, and it strips the newlines a signer wraps
  around base64. The signature is read the same way.
- **A signed pack still called its attestation section empty.** `Validate` now
  rejects a section that holds content and still carries a reason in
  `empty_sections`, in both directions. The canonical payload drops that entry
  too, so signing never changes the bytes the signature covers.
- **The action failed on a repository with no open change.** It now stops with
  a notice. Several open changes produce a warning that names the missing
  input. A change that exists and cannot be packed still fails.
- **A test read the developer's own repository.** It ran `pack` with no root,
  so it passed on a laptop that holds an untracked `.shipproof` and failed in
  continuous integration, which does not. The v0.4.0 release shipped with a red
  build for this reason. A guard test now fails when any test runs a command
  without setting a root.

### Added

- `examples/sample-project`, a small repository with one open change and one
  passing proof. The evidence workflow runs the action against it on every
  pull request.
- A `working-directory` input on the action, so it can run against a directory
  inside a repository.

### Proof

Rows P1, P2, and P3 of the definition of done are met against a live run, not
a test. Run 34284905727 uploaded a signed pack, and the local binary verifies
the keyless Sigstore signature the workflow produced. See
`docs/changes/SP-038-sample-project.md`.

## v0.4.0 — 2026-09-09

The evidence layer. This release replaces the specification tooling with a
verdict, an evidence pack, and a signature. It carries breaking changes.

**A verdict, not a report.** Every command that ends a flow prints the same
three lines: the verdict word, the count that drives it, and one next action
with a runnable command. Three verdicts exist, and no score exists.

```
VERDICT: NOT PROVEN
2 of 7 requirements have no proof. 14 changed lines match no requirement.
NEXT: run `shipproof prove SP-030` after you add a proof for R3 and R5.
```

**Three grades replace four labels.** A check now reads `observed`, `stated`,
or `claimed`. The old `derived` and `inferred` both collapse into `claimed`,
and a claimed check never proves a requirement. The report says so on the
page. `checks[].provenance` keeps the finer machine label.

**A reshaped evidence pack.** `evidence-pack.json` holds ten named sections,
including the verdict, one row per requirement with its proof result, and the
attestation. A pack is complete or absent, and `empty_sections` states why any
section is empty. An absent measurement never reads as zero.

**Signing and verification.** The pack carries a detached signature over a
canonical payload. `shipproof pack --verify <file>` returns 0 for a true
signature and 1 for an altered pack. Continuous integration signs with keyless
Sigstore; a local pack stays unsigned and says so. The verifier does not check
the Fulcio chain or the Rekor inclusion proof, and it names both as unchecked
on every run.

**Continuous integration.** `action/action.yml` builds the pack on a pull
request, signs it, uploads it, and posts one comment with the verdict block.
`.github/workflows/evidence.yml` wires it up.

**Any specification tool.** `shipproof start` reads an OpenSpec proposal, a
Spec Kit specification, or a plain Markdown list. One documented pattern finds
an obligation stated with MUST or SHALL. A pattern match is a proposal, never
a fact, and `shipproof start <id> --confirm-requirements` adopts it.

**Two commands reach a result.** `pack` runs `prove` when no fresh result
exists. `--no-prove` skips it.

**Three instruction files replace the skill catalog.** `init` installs
`capture-intent.md`, `plan-proof.md`, and `read-evidence.md` for Claude Code,
Cursor, Codex, OpenCode, and a plain `AGENTS.md`. Each one carries the rule
that matters most: an agent must never write a result that a tool did not
produce.

**A control mapping.** `docs/controls.md` maps each pack field onto the audit
question it answers and onto SOC 2 change management, EU AI Act record
keeping, and SOX change control. It states every row as a claim about
evidence. ShipProof issues no certification.

**A hook contract.** `docs/hooks.md` holds one example per harness. The
command is the contract, and a test runs every command the document names.

**An install that needs no clone.** The README leads with a one-line download.
`init` now detects a test command from a justfile, a Makefile, a
`package.json`, a `go.mod`, a `Cargo.toml`, or a `pyproject.toml`, and it
reports the one it chose.

### Breaking changes

- `schema_version` moves from `0.1` to `0.3`. A pack written by v0.3.0 no
  longer answers to the current schema. `schemas/v0.1/` and `schemas/v0.2/`
  stay on disk, and the conformance checker reads the schema a pack declares,
  so a recorded pack stays valid against its own version.
- The evidence pack renames `agent_run` to `agent`, moves `verification.checks`
  to a top-level `checks`, and removes `readiness`, `review`, and
  `agent_review`. An agent review finding is now a claimed check.
- The six skill packages are gone. `init` removes them from a repository that
  holds them.
- `phase.Result.next_skill` becomes `next_instruction` and names one of the
  three instruction files.
- The release archives drop the version from their names, so
  `releases/latest/download/shipproof_<os>_<arch>.tar.gz` resolves. A script
  that pinned `shipproof_0.3.0_<os>_<arch>.tar.gz` must change.
- The commands removed in the reduction exit 2 and print one line naming the
  replacement.

### Known limits

- The comprehension review of three readers has not run. See
  `docs/changes/SP-036-comprehension-review.md`.
- No workflow run on a sample repository has produced a pack artifact yet.
- `shipproof init` writes instruction files to `.claude/skills/`,
  `.opencode/skills/`, and `.agents/skills/`. Confirm the path your harness
  version reads.
- The portfolio report is unchanged and stays outside the definition of done.

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
