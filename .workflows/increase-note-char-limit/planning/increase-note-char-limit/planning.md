# Plan: Increase Note Char Limit

## Phase 1: Apply Change

Raise `maxNoteTextLen` from 500 to 2000, update the stale doc comment, and move the note-boundary test assertions to the new 2000/2001 boundary.

#### Tasks
status: approved

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| increase-note-char-limit-1-1 | Raise note text limit to 2000 | Reject-boundary tests (501 chars) fail as-is once the limit rises — must move to 2001; title limit (500) must stay untouched |
