# Readiness gate

A document is complete enough when it supports the next decision responsibly.

## States

- `SHAPING`: material information is still being gathered.
- `BLOCKED`: a missing decision or contradiction prevents responsible progression.
- `READY_WITH_ASSUMPTIONS`: no blocker remains, but accepted assumptions or known risks remain.
- `READY`: no blocker remains for the next stage.

## Finding classes

- `BLOCKER`: progress would be unsafe or incoherent without resolution.
- `DECISION`: the user must choose between materially different valid outcomes.
- `ASSUMPTION`: work can proceed if the assumption is recorded.
- `RISK`: known uncertainty with mitigation or conscious acceptance.
- `SUGGESTION`: optional improvement. It cannot block readiness.
- `NIT`: wording or presentation only. It cannot block readiness.

Do not create a new blocker merely because another detail could be specified.

## Blocker test

A gap is a `BLOCKER` only when at least one statement holds:

- the problem cannot be distinguished from the proposed solution;
- the target behavior has materially different valid interpretations;
- a key requirement has no evaluation method and no human acceptance method;
- two important requirements contradict each other;
- a critical dependency is assumed but not identified;
- an invented user need appears as a fact;
- the proposed scope clearly exceeds a fixed appetite.

If no statement holds, the gap is not a blocker.

## Stop condition

Stop the interview when no `BLOCKER` and no unresolved `DECISION` remains.
Declare `READY` or `READY_WITH_ASSUMPTIONS`. Report the remaining suggestions
without changing the state.

Do not create a new blocker merely because another detail could be specified.
