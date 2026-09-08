# Control mapping

ShipProof publishes one evidence pack per change. This document maps each pack
field onto the audit question it answers, and onto the control frameworks that
ask that question.

## What this document is, and what it is not

ShipProof issues no certification. It states what a tool observed, what a
person accepted, and what an agent claimed. An auditor reads the evidence and
reaches the conclusion.

Read every row below as a claim about evidence. A row states which field
answers a question. A row never states that a control is satisfied. Your
auditor decides that, and your organisation holds the control, not ShipProof.

The pack also records what it could not measure. A field that no tool reported
is absent, and `empty_sections` states why. An absent measurement never reads
as zero.

## The mapping

| Evidence field | Audit question it answers |
| --- | --- |
| `intent.snapshot_hash` and `intent.stale` | Was the approved requirement the one that shipped? |
| `requirements[].state` | Was each requirement tested before release? |
| `checks[].grade` | Is the result machine-observed or asserted? |
| `unexplained_change.line_findings` | Did the change stay inside the approved scope? |
| `agent.model` and `agent.session_id` | Which model produced the change, and under which session? |
| `attestation.signature` | Can the record be altered after the fact? |

## Row by row

### Was the approved requirement the one that shipped?

`intent.source_path` names the intent document. `intent.snapshot_hash` holds
the SHA-256 of that document at the moment `shipproof start` ran.
`intent.captured_at` holds that moment. `intent.stale` reports whether the
document changed after the snapshot, and `intent.current_source_hash` holds the
hash of the document as it now stands.

A `false` value in `intent.stale` states that the document did not change. A
`true` value states that it did, and the pack still reports both hashes, so a
reader can compare them.

- SOC 2 change management: the authorised request behind the change.
- EU AI Act record keeping: the specification that the system was built to.
- SOX change control: the approved requirement that entered the release.

### Was each requirement tested before release?

`requirements[]` holds one row per requirement. `requirements[].id` names it.
`requirements[].proof_refs` names the commands that judge it.
`requirements[].state` reports the outcome, and `requirements[].detail` states
the reason in one sentence.

A requirement with no proof reads `unproven`. It never reads as passed.

- SOC 2 change management: the test evidence for the change.
- EU AI Act record keeping: the verification of the stated requirement.
- SOX change control: the test result that supports the release.

### Is the result machine-observed or asserted?

`checks[].grade` holds one of three values.

- `observed`. A tool ran and a machine recorded the result.
- `stated`. A person accepted it and signed for it.
- `claimed`. An agent or a person asserted it, and no tool confirmed it.

A `claimed` check never proves a requirement. `checks[].provenance` holds the
finer machine label behind the grade, and `checks[].detail` states the reason.

- SOC 2 change management: the reliability of the evidence itself.
- EU AI Act record keeping: the separation of a measurement from an assertion.
- SOX change control: the independence of the test result.

### Did the change stay inside the approved scope?

`unexplained_change.line_findings` names every changed line that no approved
proof ran. `unexplained_change.file_findings` names every changed file that no
proof names. `unexplained_change.uninstrumented_lines` counts the changed lines
that the coverage command could not reach.

Read `unexplained_change.measured` first. A `false` value states that ShipProof
did not reach the measurement, and the counts then state nothing.
`unexplained_change.coverage_available` reports whether a coverage command ran
at all.

- SOC 2 change management: the scope of the change against the request.
- EU AI Act record keeping: the change that reached the system.
- SOX change control: the unauthorised change that entered a release.

### Which model produced the change, and under which session?

`agent.provider`, `agent.model`, and `agent.agent_version` name the tool.
`agent.session_id` names the session, and `agent.raw_log_ref` points to the
recorded log. `agent.started_at` and `agent.ended_at` bound the work.

Every field here comes from the telemetry that the tool wrote. A field that the
tool did not report is absent from the pack.

- SOC 2 change management: the identity that made the change.
- EU AI Act record keeping: the automated system that produced the output, and
  the retention of its log.
- SOX change control: the actor behind the change.

### Can the record be altered after the fact?

`attestation.signature` holds the signature over a canonical serialization of
the pack. `attestation.certificate` holds the signing certificate, and
`attestation.subject` names the revision the pack covers.
`attestation.digest` holds the SHA-256 of the signed payload.

Run `shipproof pack --verify <file>`. It returns 0 for a pack that matches its
signature and 1 for a pack that changed after the signature.

A local run produces no signature, and `empty_sections` says so. Only a
build-system pack is audit-grade.

The verifier checks the signature over the payload and it reports the
certificate subject. It does not check the Fulcio certificate chain, and it
does not check the Rekor inclusion proof. It names both as unchecked on every
run.

- SOC 2 change management: the integrity of the retained evidence.
- EU AI Act record keeping: the tamper evidence on the retained log.
- SOX change control: the integrity of the audit record.

## How to hand a pack to an auditor

1. Open the pull request. The workflow writes the pack and posts the verdict.
2. Download the build artifact `evidence-pack.json`.
3. Run `shipproof pack --verify evidence-pack.json`.
4. Give the auditor the pack, this document, and the output of step 3.
