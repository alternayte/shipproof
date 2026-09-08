# Benchmarks

This directory holds repeatable public benchmark material.

```text
benchmarks/
  tasks/            Implementation benchmark tasks. Each task has task.md,
                    starting-commit.txt, hidden-evaluation/, and
                    expected-properties.yaml.
  runs/             Recorded implementation benchmark runs. Failed runs stay.
```

## Rules

- Benchmark tasks start from fixed commits. The agent never opens
  `hidden-evaluation/`.
- Every condition runs at least three times when practical. Run order is
  randomized when practical.
- Failed runs remain in the dataset. Do not delete them.
- Unknown fields stay unknown. Do not estimate provider cost.
