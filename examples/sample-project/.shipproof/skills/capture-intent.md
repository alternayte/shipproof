# Capture the intent

Record the document that states what the change must do. ShipProof compares
the shipped diff against this record, so the record is the anchor of every
evidence pack.

**An agent must never write a result that a tool did not produce.**

## The command

```
shipproof start <change-id> --intent <path>
```

`start` reads the document, computes its SHA-256, and stores the hash with the
capture time. It writes no requirement that the document does not hold.

## What counts as an intent document

Any document that a specification tool produced. ShipProof reads the file. It
does not care which tool wrote it.

- An OpenSpec change proposal.
- A Spec Kit specification.
- A plain Markdown requirements list.
- A ticket exported to a file.

## How to record the requirements

Give each requirement a stable identifier and one sentence that states the
obligation. Take both from the document. Do not invent a requirement that the
document does not state, and do not drop one that it does.

A requirement that the document leaves unclear stays unclear. Write the
sentence the document holds, and raise the ambiguity with the person who owns
the document. Do not resolve it on your own and record the resolution as the
intent.

## When the document changes

`start` records the hash of the document as it stood. A later edit makes the
record stale, and `shipproof status` reports that. Re-run `start` with
`--force` after you agree the new document is the intent.

A proof that ran against the older document proves nothing about the document
that now stands. Never report such a proof as current.

## What you must not do

- Do not write a requirement identifier that the document does not carry.
- Do not record a requirement as captured when you could not read the file.
- Do not edit the intent document to match the code you wrote.
