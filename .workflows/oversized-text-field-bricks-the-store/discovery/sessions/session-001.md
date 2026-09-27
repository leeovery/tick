# Discovery Session 001

Date: 2026-09-27
Work unit: oversized-text-field-bricks-the-store

## Description (as of session)

A task record whose JSONL line passes ~64 KiB is accepted and written, after which every store-reading command (rebuild included) fails, while `tick doctor` reports the project healthy — bound what can be written and make doctor fail on a store it could not fully read.

## Seed

- seeds/2026-09-21-oversized-text-field-bricks-the-store.md (inbox:bug)

## Imports

(none)

## Map State at Start

(n/a — single-topic work)

## Exploration

Opened from the inbox bug report: a description large enough to push its JSONL line past 64 KiB is saved without complaint, after which `list`, `stats` and `rebuild` all fail with `bufio.Scanner: token too long`, leaving hand-editing the JSONL as the only recovery; `tick doctor` meanwhile reports `JSONL syntax: OK` and no issues, because its reader drops the overlong line without checking the scanner's error.

The user confirmed both halves belong in this one piece of work — the write-side limit and the doctor's silent pass — and asked whether fixing one removes the other. Answer given: no, they are independent. A write-time cap stops tick producing an unreadable line going forward, but does not rescue a store that already holds one (written before the cap ships), does not cover lines produced outside the capped write path (an older tick binary, hand edits, other tools), and the doctor's silent pass hides all of those alike. The doctor half holds regardless of where any cap lands.

A further thread surfaced for investigation: a cap on the description field alone does not bound the line. Notes are capped at 2000 characters each but their count is unbounded, so roughly 33 full-length notes push a task past the reader's ceiling today with no description at all; titles are uncapped too, and transitions accumulate. The limit therefore has to be reasoned about per record (the line the reader consumes), not only per field — or the reader's ceiling itself reconsidered. The inbox report's agreed direction (no support for arbitrarily large text; a generous but finite limit enforced on the write path with a clear error, consistent with the existing note cap) stands as the starting shape, subject to investigation. Recovery for stores already over the ceiling is an open question for investigation.

Work type settled as bugfix: a successful save that then locks every other command is a reproducible malfunction with a specific error, and the health check's false "healthy" is a second one; both hinge on the same question of when a line becomes unreadable.

## Edits

(none)

## Topics Identified

(none)

## Conclusion

Routed to investigation.
