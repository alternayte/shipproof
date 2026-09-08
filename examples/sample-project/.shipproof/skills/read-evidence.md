# Read the evidence and report the verdict

The evidence pack states what happened. Your job is to report it without
inflating it.

**An agent must never write a result that a tool did not produce.**

## The commands

```
shipproof status <change-id>
shipproof pack <change-id>
shipproof pack --verify <file>
```

`status` prints the verdict. `pack` writes the evidence pack and the report.
`pack --verify` checks a signature.

## The verdict

Three verdicts exist, and no score exists.

- `PROVEN`. Every requirement carries a passing observed proof or an accepted
  human proof, and no unexplained change remains.
- `NOT PROVEN`. The work is incomplete or unproven.
- `FAILED`. A proof ran and failed, or the gate failed.

Report the verdict word that the pack holds. Do not soften `NOT PROVEN` into
"mostly done". Do not describe `FAILED` as a minor issue. Do not report
`PROVEN` because the tests you remember running passed.

## Unexplained change

The pack counts the changed lines that no requirement explains. Read
`unexplained_change.measured` first.

A `false` value means that ShipProof did not reach the measurement. The count
then states nothing. Report it as not known. Never report it as zero.

## Absence

A field that no tool reported is absent from the pack, and `empty_sections`
states why. Report the absence and its reason. Do not fill a missing field
with a guess, and do not omit a section in silence because it was empty.

## The signature

A local pack is unsigned. Only a pack that the build system produced is
audit-grade, and `pack --verify` checks it.

`--verify` checks the signature over the payload. It does not check the Fulcio
certificate chain, and it does not check the Rekor inclusion proof. Report that
limit when you report the result.

## What you must not do

- Do not state a verdict that the pack does not hold.
- Do not report a requirement as proven when its grade is `claimed`.
- Do not report a count that no tool measured.
- Do not summarise a pack you did not read.
