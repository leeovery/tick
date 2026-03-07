TASK: cli-enhancements-4-7 -- "Note add subcommand"

ACCEPTANCE CRITERIA:
- `tick note add <id> <text>` appends a timestamped note; text from remaining args after ID; validates non-empty and max 500 chars
- Adding a note updates the task's `Updated` timestamp
- Edge cases: missing id, missing text, text from multiple remaining args, task not found

STATUS: Complete

SPEC CONTEXT:
Specification defines `tick note add <id> <text>` as appending a timestamped Note (Text + Created) to a task. Text is multi-word from remaining args after ID. Validation: empty text errors, max 500 chars. Adding a note updates the task's Updated timestamp. Notes are stored in JSONL as JSON array with omitempty and in SQLite via task_notes table. No separate `note list` command -- notes are viewed via `tick show`.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/note.go:39-93 (RunNoteAdd function)
  - /Users/leeovery/Code/tick/internal/cli/note.go:14-35 (handleNote routing)
  - /Users/leeovery/Code/tick/internal/cli/app.go:106-107 (App.Run dispatch)
  - /Users/leeovery/Code/tick/internal/task/notes.go:52-61 (ValidateNoteText)
  - /Users/leeovery/Code/tick/internal/task/notes.go:64-66 (TrimNoteText)
- Notes:
  - Correctly parses positional args by skipping flag-like args (prefixed with "-")
  - Joins remaining args after ID with spaces to form the note text
  - Trims whitespace via TrimNoteText before validation
  - Validates non-empty and max 500 chars (using utf8.RuneCountInString for correct Unicode handling)
  - Resolves partial IDs via store.ResolveID
  - Appends Note with UTC timestamp truncated to seconds
  - Updates task's Updated timestamp to the same `now` value
  - Uses store.Mutate for safe concurrent writes with file locking
  - Outputs the task detail after mutation via outputMutationResult
  - Follows the established handler signature pattern: `RunNoteAdd(dir, fc, fmtr, args, stdout)`

TESTS:
- Status: Adequate
- Coverage:
  - Happy path: adds a note to a task (line 31)
  - Multi-word text: collects text from multiple remaining args (line 59)
  - Missing ID: errors with usage hint (line 87)
  - Missing text: errors with "note text is required" (line 99)
  - Text exceeds 500 chars: errors with "exceeds maximum length" (line 115)
  - Task not found: errors with "not found" (line 132)
  - Partial ID resolution: resolves short prefix (line 144)
  - Updated timestamp: verifies task's Updated field is refreshed (line 172)
  - Note timestamp: verifies note's Created field is set to current time (line 200)
  - No sub-subcommand: errors with usage hint (line 511, TestNoteNoSubcommand)
  - Domain-level tests in /Users/leeovery/Code/tick/internal/task/notes_test.go cover validation edge cases (empty, whitespace-only, exactly 500, 501 chars, trimming)
- Notes: Tests are well-structured with t.Run subtests, use t.Helper on the runNote helper, and verify both persistence (readPersistedTasks) and error output. All planned edge cases are covered. No over-testing observed -- each test covers a distinct behavior.

CODE QUALITY:
- Project conventions: Followed. Uses handler signature pattern, stdlib testing only, t.Run subtests, t.TempDir via helpers, error wrapping with fmt.Errorf, DI via App struct fields.
- SOLID principles: Good. Validation is in the domain layer (task package), CLI command logic is in the CLI layer, storage access via Store. Single responsibility is clean.
- Complexity: Low. Linear flow: parse args, validate, open store, resolve ID, mutate, output. No nested conditionals or complex branching.
- Modern idioms: Yes. Uses range over index for slice mutation, defer for store cleanup, UTC truncation for timestamp consistency.
- Readability: Good. Clear function name, doc comment, straightforward flow. The positional arg parsing with flag-skipping is commented.
- Issues: None identified.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The flag-skipping logic (lines 42-46) that filters out args starting with "-" is a simple heuristic. If a note text word started with a dash (e.g., "task needs -verbose flag"), it would be silently dropped. However, this is consistent with how other commands in the codebase handle args, and in practice CLI users would quote such text. This is an edge case that could be documented but is not a blocking concern.
