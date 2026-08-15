---
name: benchmark-run
description: Run one ShipProof benchmark task from a fixed starting commit and record the result without exposing hidden evaluation material. Use when evaluating coding agents, workflow variants, or skill versions against baseline conditions. Never open hidden-evaluation content, never delete failed runs, and never estimate cost as observed fact.
metadata:
  shipproof-version: "0.2"
---

# Run one benchmark task

A benchmark compares one condition against a fixed task. Use this skill to
run a single task under one condition. The comparison itself comes later from
the recorded runs.

## Inputs

Each task directory contains:

```text
benchmarks/tasks/<task>/
  task.md
  starting-commit.txt
  hidden-evaluation/
  expected-properties.yaml
```

The agent runs the task in a clean worktree at the fixed starting commit.

## Procedure

1. Read `task.md` and `starting-commit.txt`.
2. Do not open any file under `hidden-evaluation/`.
3. Reset the worktree to the starting commit.
4. Run the task under the assigned condition:
   - `naive`: the task prompt only;
   - `instructions`: the task prompt plus the repository AGENTS.md;
   - `shipproof`: the task prompt plus the full ShipProof workflow.
5. Record metrics in `benchmarks/runs/<task>/<condition>-run-<n>.json`.

## Record schema

```json
{
  "schema_version": "0.1",
  "task": "webhook-retry",
  "condition": "shipproof",
  "agent": "claude-code",
  "agent_version": "1.0.0",
  "model": "unknown",
  "starting_commit": "abc1234",
  "completion_result": "pass",
  "hidden_test_result": "unknown",
  "regressions": 0,
  "agent_elapsed_seconds": 0,
  "human_intervention_seconds": 0,
  "review_seconds": 0,
  "observed_cost": null,
  "iterations": 1,
  "changed_files": 0,
  "changed_lines": 0,
  "verification_failures": 0
}
```

## Rules

- Preserve unknown information as unknown. Do not estimate provider cost.
- Keep failed runs in the dataset. Never delete a run because it failed.
- The hidden test result is unknown while `hidden-evaluation/` remains unopened.
- Run each condition at least three times when practical.
- Randomize run order when practical.

## Record the run

```bash
shipproof skill eval record <case-id> --condition <without|previous|candidate> --file <result.json>
```

Use that command for skill eval runs. Use the benchmark run schema above for
implementation benchmark tasks.
