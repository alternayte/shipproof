# SP-022 — Unexplained-change signal

## Problem

ShipProof proves each requirement, but a proof can pass while it touches only
part of the changed code. No signal names the changed code that no approved
proof ran. A reviewer cannot see, from the evidence pack alone, which lines a
change moved and no test observed.

## Desired outcome

ShipProof reports the changed code that no proof ran. The report works at
line level when a coverage command is configured, and it degrades to file
level when it is not. The report never fails a change. It gives the reviewer
a count of the lines it could not judge, and it feeds the evidence pack and
the review packet.

## Scope

A coverage profile parser reads a merged Go coverage profile. A changed-hunk
collector reads Git diff hunks. An unexplained-change renderer combines the
two into a line-level or file-level report. The evidence pack gains one check
per requirement, and the review packet leads with the unexplained-change
section.

## Requirements

### SP-022-R1 — Coverage profile parser

A coverage profile parser reports executed, not executed, and not
instrumented. A line absent from every instrumented block carries no claim.

### SP-022-R2 — Changed-hunk collector

The changed-hunk collector returns the post-image line ranges and the
best-effort symbol from the Git hunk header.

### SP-022-R3 — Line-level report

The unexplained report works at line level when a coverage command is
configured. Its provenance reads `observed`.

### SP-022-R4 — File-level report

The unexplained report degrades to file level when no coverage command is
configured. Its provenance reads `derived`, and it states that it cannot see
inside a file.

### SP-022-R5 — Unjudged-line count

The report states the count of changed lines it could not judge.

### SP-022-R6 — Evidence and review integration

The evidence pack holds one check per requirement, and the review packet
leads with the unexplained-change section.
