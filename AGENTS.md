# ShipProof repository instructions

Read the ShipProof instruction file that matches the task. Three exist:
`capture-intent.md`, `plan-proof.md`, and `read-evidence.md`. They live in
`.shipproof/skills/`.

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

## Self-hosted workflow: on

ShipProof runs its own workflow on itself. v0 closed, and the condition that
paused this is met.

Start every change with the tool:

```
shipproof start SP-0NN --intent docs/changes/SP-0NN-<slug>.md
shipproof start SP-0NN --confirm-requirements
shipproof status SP-0NN
shipproof pack SP-0NN
```

Write the change document first, in the SP-039 format, with a `### SP-0NN-RN —`
heading per requirement so the native pattern reads it. Keep one independently
verifiable change per session.

`.shipproof/changes/SP-001` to `SP-020` are a historical record of the v0 line.
Read them. Do not edit them.

Report what the tool reported. A verdict that the tool did not print is a
claim, and this repository holds itself to the rule it ships:

> An agent must never write a result that a tool did not produce.

`docs/design/shipproof-sdd.md` is the canonical contract. Section 14 holds the
complete definition of done. Section 15 holds the open decisions.

- A criterion is met when its proof command succeeds. Nothing else changes its
  state.
- A suggestion or a nit must never reopen a met criterion.
- A useful idea that no criterion covers becomes a new document under
  `docs/changes/`.

Read `docs/changes/` for the change backlog.
