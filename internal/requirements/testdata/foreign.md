# Payment retry specification

Some prose that names no requirement.

## Retry a failed charge

The gateway returns a transient error.

## Cap the retry count

- MUST stop after five attempts.
- Prefer an exponential delay.

### Record every attempt

Write one row per attempt.
