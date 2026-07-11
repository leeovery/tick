# Discovery Session 001

Date: 2026-07-11
Work unit: increase-note-char-limit

## Description (as of session)

Increase note text character limit from 500 to 2000.

## Seed

(none)

## Imports

(none)

## Map State at Start

(n/a — single-topic work)

## Exploration

The user reported that the note text character limit in Tick (currently
500) has proven too low in real use, and proposed raising it to 2000.
They were also curious whether the main task description field carries a
numeric character cap.

Checked the code: the description field has no length limit — it only
rejects empty/whitespace on update (`ValidateDescriptionUpdate`,
`internal/task/task.go`). Two fields are capped, both at 500: note text
(`maxNoteTextLen`, `internal/task/notes.go:12`) and title
(`maxTitleLen`, `internal/task/task.go:33`).

Confirmed scope with the user: the change is limited to the note text
limit only (500 → 2000). No change to the description (left unbounded by
design) and no change to the title limit. The description question was a
curiosity, not a work item.

This is a small, mechanical change to a single constant with no behaviour
debate and nothing to diagnose — a quick-fix. Routes to scoping, then
implementation and review.

## Edits

(none)

## Topics Identified

(none)

## Conclusion

Routed to scoping.
