TASK: cli-enhancements-4-9 -- Notes display in show output and all formatters

ACCEPTANCE CRITERIA:
- Show output displays notes; notes shown chronologically (most recent last) with timestamp format YYYY-MM-DD HH:MM
- All three formatters updated for notes display in detail views

STATUS: Complete

SPEC CONTEXT:
The specification (Notes section) defines that notes in `tick show` are shown chronologically (most recent last) with format:
```
Notes:
  2026-02-27 10:00  Started investigating the auth flow
  2026-02-27 14:30  Root cause found -- token refresh race condition
```
Edge cases from the task: task with no notes, task with multiple notes, note with long text.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/show.go:30` -- `showData` struct includes `notes []task.Note`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:143-164` -- `queryShowData` queries notes from `task_notes` table with `ORDER BY created ASC` (chronological, most recent last)
  - `/Users/leeovery/Code/tick/internal/cli/show.go:238` -- `showDataToTaskDetail` passes notes through to `TaskDetail`
  - `/Users/leeovery/Code/tick/internal/cli/format.go:103` -- `TaskDetail` struct includes `Notes []task.Note`
  - `/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:133-138` -- PrettyFormatter renders notes with `YYYY-MM-DD HH:MM` timestamp format (`"2006-01-02 15:04"`)
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:101-102` -- ToonFormatter renders notes section (always present, even with count 0)
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:215-227` -- `buildNotesSection` renders notes as TOON tabular format with `{text,created}` schema
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:49-52` -- `jsonNote` struct with `text` and `created` fields
  - `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:93-99` -- JSONFormatter converts notes to `jsonNote` with ISO 8601 timestamps
- Notes:
  - PrettyFormatter uses `YYYY-MM-DD HH:MM` timestamp format as specified by the acceptance criteria
  - ToonFormatter uses ISO 8601 (`2006-01-02T15:04:05Z`) via `task.FormatTimestamp()` -- appropriate for machine-oriented format
  - JSONFormatter uses ISO 8601 via `task.FormatTimestamp()` -- standard for JSON
  - Chronological ordering enforced by SQL `ORDER BY created ASC`
  - Pretty/JSON omit notes section when empty; Toon shows `notes[0]{text,created}:` (consistent with blocked_by/children pattern)

TESTS:
- Status: Adequate
- Coverage:
  - Integration tests in `/Users/leeovery/Code/tick/internal/cli/list_show_test.go`:
    - Line 631: "it displays notes in show output when task has notes" -- verifies Pretty output with multiple notes, checks `YYYY-MM-DD HH:MM` timestamp format and chronological ordering
    - Line 659: "it omits notes section in show when task has no notes" -- edge case: no notes
  - ToonFormatter unit tests in `/Users/leeovery/Code/tick/internal/cli/toon_formatter_test.go`:
    - Line 608: "it displays notes in toon show output" -- verifies multiple notes with timestamps in ISO 8601
    - Line 645: "it shows empty notes in toon when no notes" -- verifies `notes[0]{text,created}:` empty section
    - Line 107-111: "it formats show with all sections" -- verifies notes section appears as 4th section with `notes[0]{text,created}:`
  - PrettyFormatter unit tests in `/Users/leeovery/Code/tick/internal/cli/pretty_formatter_test.go`:
    - Line 643: "it displays notes in pretty show output with timestamps" -- single note with YYYY-MM-DD HH:MM format
    - Line 670: "it omits notes section in pretty when no notes" -- edge case
    - Line 691: "it displays multiple notes chronologically in pretty" -- multiple notes, verifies chronological order
    - Line 719: "it displays note with long text without truncation" -- edge case: long text
  - JSONFormatter unit tests in `/Users/leeovery/Code/tick/internal/cli/json_formatter_test.go`:
    - Line 939: "it displays notes in json show output with text and created" -- two notes with text and ISO 8601 timestamps
    - Line 990: "it shows empty notes array in json when no notes" -- empty array not null
  - All three edge cases covered: no notes (all three formatters), multiple notes (pretty, toon, JSON), long text (pretty)
- Notes: Test coverage is thorough and balanced. Each formatter has both positive and empty/negative tests. Edge cases from the task are all addressed.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, t.Helper on helpers, error wrapping with fmt.Errorf
- SOLID principles: Good -- each formatter handles its own rendering independently; shared data structures (TaskDetail, Note) provide clean contracts
- Complexity: Low -- straightforward rendering logic, no complex control flow
- Modern idioms: Yes -- Go generics used in `encodeToonSection`, proper use of `strings.Builder`
- Readability: Good -- code is self-documenting, consistent patterns across formatters
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The Toon and JSON formatters use ISO 8601 timestamps for notes rather than the `YYYY-MM-DD HH:MM` format specified in the spec example. This is a reasonable design decision since those formats are machine-oriented, and ISO 8601 is the standard for both. The spec example format is only shown in the context of human-readable display (which Pretty matches exactly). No change needed.
