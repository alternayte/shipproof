# Retry transient webhook deliveries

## Problem
Customers lose webhook deliveries when a destination has a short outage.

## Users and desired outcome
Customers receive webhook deliveries after a transient destination failure.
Success means a delivery completes without a manual replay.

## Scope and appetite
Retry transient HTTP failures. Do not add customer-configurable retry policies.

## Requirements and acceptance
R-01: Retry a 503 response.
Verify that a later successful response completes the delivery.

## Assumptions, risks, and unknowns
Assume destinations can receive duplicate requests. Use delivery identifiers for deduplication.
