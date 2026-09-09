# SP-040 — Make adding and tracking a requirement easy

Status: proposed. Not started.
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

## Open questions to settle before building

1. Does a person confirm requirements one at a time, or is the whole set the
   unit? The whole set is simpler and it forces an all-or-nothing decision on
   a long document.
2. What happens when the document gains a requirement after `start`? A merge
   is the useful answer and the harder one. `--force` is what exists.
3. Should ShipProof check a proof command before it records it, so a typo
   reports a mistake rather than a failure?
4. Where does the list of requirements without a proof belong? `status` is the
   natural home, and its output is currently short.

## Not a defect in the report

Row U1 stays met, and `SP-039` stays met. The page explained itself; the tool
behind it is the thing to improve. Recording this as a change document, rather
than as a report fix, keeps that distinction honest.
