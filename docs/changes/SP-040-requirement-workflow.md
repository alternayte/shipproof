# SP-040 — Make adding and tracking a requirement easy

Status: implemented.
Source: the second reader review of 2026-09-09, recorded in
`docs/changes/SP-036-comprehension-review.md`.

## Problem

The second reader review left two questions. One was a wording defect and it
is fixed. This is the other, and it is not a wording defect.

Readers understood what a requirement was, what a proof was, and why an
unproven requirement matters. They then asked how a person would add and track
a requirement in practice.

The honest answer today is that it is clumsy.

- `start` proposes requirements from one documented pattern, and a person
  confirms the whole set. There is no way to accept some and reject others.
- A requirement added to the document after `start` needs `start --force`,
  which re-snapshots everything.
- A proof is added by hand-editing `verification.json`. Nothing checks the
  command before it runs, so a typo becomes a failing proof rather than a
  message.
- Nothing lists the requirements that still need a proof, except the report.
- No command removes a requirement, renames one, or marks one as no longer in
  scope.

The reader asked a workflow question. The report answered it correctly and the
tool then made it hard.

## The constraint that governs any fix

The same review praised the page for not being wordy. Every reader in round two
read it without complaint about length.

A fix here must not answer this question by adding prose to the report. The
report already says where a requirement came from and how to add a proof. The
work belongs in the commands.

Section 4 also fixes the surface at five commands plus two support commands. A
fix must live inside those, as an option or a subcommand of an existing verb.

## The four questions, settled

**1. Is the whole set the unit of confirmation?** Yes. A proposal is a readable
JSON file, and a person who wants a subset edits it before they confirm. That
needs no new surface, and SP-034 narrowed the pattern to obligations, so a
proposal now carries little to prune. `--confirm-requirements` prints what it
adopted, so the person sees the result of their edit.

**2. What happens when the document gains a requirement?** `start --force`
merges. It adds what the document gained and keeps every requirement that
stands, with its confirmation. It never deletes. A requirement that the
document no longer states is reported and left in place, because principle 6
of Section 2 says nothing is deleted to make a change pass. A person removes
it deliberately or not at all.

This was the important one. It was not an ergonomic gap, it was a correctness
bug: `--force` re-snapshotted the document and left the requirement set stale,
so a pack reported a fresh intent against requirements the document no longer
matched.

**3. Does ShipProof check a proof command?** It reports one it cannot find.
A command whose program is not on the PATH is a broken proof, not a failed
requirement, and the two must never read the same. `status` names it before
`prove` runs and records a failure that means something else.

**4. Where does the list of unproven requirements live?** `status`. It is the
fast loop, its output is short, and the report already carries the full table.

## What was built

- `start --force` merges the requirement set with the document. It reports
  what it added and what the document no longer states.
- `status` names every requirement with no proof, and every planned proof
  whose program it cannot find.
- The `capture-intent` instruction file states that a proposal is editable
  before it is confirmed.

## Not a defect in the report

Row U1 stays met, and `SP-039` stays met. The page explained itself; the tool
behind it is the thing to improve. Recording this as a change document, rather
than as a report fix, keeps that distinction honest.
