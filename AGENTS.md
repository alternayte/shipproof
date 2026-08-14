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

For shaping work:

- Persist durable shaping state under `.shipproof/shaping/`.
- Run `shipproof shape check` after editing shaping state.
- Stop the interview when the finite readiness gate is satisfied.

For material implementation work:

- Read `.shipproof/changes/<change-id>/verification.json` when present.
- Run `shipproof verification check <change-id>` before implementation.
- Keep one independently verifiable change per implementation session where practical.

Read `docs/design/NEXT.md` for the current ordered dogfood backlog.
