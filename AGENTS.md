# ShipProof repository instructions

Use the smallest ShipProof skill that matches the task.

Follow these invariants:

- Read the approved intent before implementation.
- Keep changes bounded to the approved scope.
- Preserve unknown information as unknown.
- Prefer deterministic evidence for facts that tools can measure.
- Never weaken verification to make a change pass.
- Do not add architecture or dependencies without a contextual reason.
- Use STE-assisted technical prose for generated specifications, findings, and evidence summaries.
- Run `just verify` before declaring implementation complete.
- Treat suggestions and nits as non-blocking.

## Self-hosted workflow: paused

ShipProof does not currently run its own workflow on itself. Do not create a
shaping session, an intent snapshot, a verification plan, or an evidence pack
for work in this repository.

Write a change document under `docs/changes/` in the SP-011 format instead.
Keep one independently verifiable change per implementation session.

The existing `.shipproof/` state is a historical record of SP-001 to SP-020.
Read it. Do not extend it.

Self-hosted work resumes after v0 closes. See Section 26 of the SDD.

`docs/design/shipproof-v0-sdd.md` is the canonical v0 contract. Section 24 holds the complete definition of done. Section 25 holds the closure rules.

- A criterion is met when its proof command succeeds. Nothing else changes its state.
- A suggestion or a nit must never reopen a met criterion.
- A useful idea that no criterion covers becomes a new document under `docs/changes/`. It is not v0 work.

Read `docs/changes/` for the change backlog.
