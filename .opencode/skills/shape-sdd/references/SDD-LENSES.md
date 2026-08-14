# Conditional SDD review lenses

Apply a lens only when the design contains that concern.

## Persistent data
Ownership, invariants, consistency, migrations, rollback, and retention when relevant.

## Messaging or distributed work
Delivery semantics, ordering, idempotency, retries, poison work, and recovery.

## Public or shared API
Contract, compatibility, versioning, authorization, and error behavior.

## Concurrency
Races, coordination, invariants, retry safety, and cancellation where relevant.

## Sensitive data
Trust boundaries, authorization, privacy, secrets, and audit requirements.

## Background processing
Retries, duplicate work, cancellation, timeouts, and recovery.

## Production operation
Observability, deployment, rollback, capacity constraints, and operational recovery.

Absence of an irrelevant section is not a defect.
