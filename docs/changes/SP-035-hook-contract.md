# SP-035 — The hook contract

Status: implemented.
Source: `docs/design/shipproof-sdd.md`, Section 8.1, Section 14.5.
Definition of done: row T3.

## Problem

Row T3 says that the hook contract runs a command and never a library call,
and that a documented hook example exists per harness with a test on the
example command.

No hook document exists. A user who wants ShipProof to run at the end of a
session has nothing to copy, and nothing states what ShipProof guarantees to a
hook author.

## Scope

Add `docs/hooks.md`. It states the contract, and it holds one example per
harness. Add a test that runs every command the document names.

## What the test can prove, and what it cannot

The command is the contract, so the test proves the command.

- It proves that every command in the document is a real ShipProof command.
- It proves that every command names only the surface of Section 4.
- It proves that each command exits with a code the document states.
- It proves that no example calls a Go package.

It does not prove that a given harness parses the configuration around the
command. That format belongs to the harness, it changes with the harness
version, and ShipProof cannot verify it offline. The document must say so
rather than imply a guarantee that no proof supports.

## Requirements

R1. `docs/hooks.md` states the contract: a hook runs a command, and never a
library call.

R2. The document holds one example for each of the five harness targets:
Claude Code, Cursor, Codex, OpenCode, and a plain `AGENTS.md`.

R3. Every example runs `shipproof prove` or `shipproof pack`, as Section 8.1
states.

R4. Every command in the document names only the surface of Section 4.

R5. The document states the exit code that each command returns, and what a
hook author must do with it.

R6. The document states plainly that ShipProof verifies the command and not
the harness configuration format.

R7. No example imports a ShipProof package or calls a Go function.

## Acceptance

- A test reads every command from `docs/hooks.md` and runs it against a
  temporary repository.
- A test asserts that each command exits with the documented code.
- A test asserts that every harness target has an example.
- A test asserts that the document names no command outside Section 4.
- A test asserts that no example holds a Go import path.
- `just verify` exits 0.

## Non-goals

- Installing a hook. Section 8.1 says the hook is optional, so `init` must not
  write one.
- Any new command or flag. The surface of Section 4 does not change.
