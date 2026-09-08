# Plan a proof for each requirement

Map every requirement onto something that judges it. A requirement with no
proof is unproven, and ShipProof reports it as unproven.

**An agent must never write a result that a tool did not produce.**

## The command

```
shipproof prove <change-id>
```

`prove` runs the repository gate, then runs each proof on its own, and records
one result per proof with its exit code.

## What a proof is

A proof is a command that exits 0 when the requirement holds and non-zero when
it does not. A test, a lint rule, a schema check, and a build all qualify.

Write the narrowest command that judges the requirement. `go test -run
TestRetryIsIdempotent ./internal/queue` judges one requirement. `go test
./...` judges the suite, and it tells a reader nothing about which requirement
passed.

## When no command can judge it

Some requirements need a person. A visual layout, a wording choice, and a
policy judgement all need a person.

Mark that proof as human. A human proof stays unproven until a person accepts
it and their name and the time are recorded. ShipProof then grades it
`stated`.

Do not invent a command that appears to judge such a requirement. A command
that passes for the wrong reason is worse than an honest human proof.

## Grades

Every result carries one of three grades.

- `observed`. A tool ran and a machine recorded the result.
- `stated`. A person accepted it and signed for it.
- `claimed`. Someone asserted it and no tool confirmed it.

A `claimed` result never proves a requirement. If you write a summary of your
own work, it is `claimed`, whatever you believe about its accuracy.

## What you must not do

- Do not record a proof result that you did not run.
- Do not widen a failing proof until it passes.
- Do not delete a proof to make a change pass. A failing proof stays visible.
- Do not mark a proof human because the command is hard to write.
