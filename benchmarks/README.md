# Benchmarks

This directory holds repeatable public benchmark material.

```text
benchmarks/
  tasks/            Implementation benchmark tasks. Each task has task.md,
                    starting-commit.txt, hidden-evaluation/, and
                    expected-properties.yaml.
  runs/             Recorded implementation benchmark runs. Failed runs stay.
  skill-evals/      Recorded skill eval runs, organized by case and condition.
                    Conditions: without, previous, candidate.
```

## Rules

- Benchmark tasks start from fixed commits. The agent never opens
  `hidden-evaluation/`.
- Every condition runs at least three times when practical. Run order is
  randomized when practical.
- Failed runs remain in the dataset. Do not delete them.
- Unknown fields stay unknown. Do not estimate provider cost.

## Skill evals

Each skill eval run is recorded with:

```bash
shipproof skill eval record <case-id> --condition <without|previous|candidate> --file result.json
```

Show recorded runs:

```bash
shipproof skill eval results
shipproof skill eval results <case-id> --regression
```

A regression is reported when a candidate run fails a task a baseline run
passed, recalls fewer blockers, or raises the false blocker rate.

The eval case definitions live in `skills/evals/spec-skills.cases.json`.
