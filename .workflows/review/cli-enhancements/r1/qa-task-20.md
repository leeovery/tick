TASK: cli-enhancements-4-5 -- Note data model with validation and JSONL serialization

ACCEPTANCE CRITERIA:
- Task struct has Notes []Note field with json:"notes,omitempty"
- Note has Text string and Created time.Time
- Note text validation: non-empty, max 500 chars
- Edge cases: empty note text, note exactly 500 chars, 501-char note, whitespace-only text

STATUS: Complete

SPEC CONTEXT:
The specification defines Notes as "timestamped text entries appended to a task -- a log of context, decisions, progress." The Note data model is {Text string, Created time.Time}. Validation requires non-empty text and max 500 chars. JSONL storage uses JSON array with omitempty. Adding/removing notes updates the task's Updated timestamp (handled by later tasks 4-7/4-8).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/task/notes.go:15-18 -- Note struct with Text and Created fields
  - /Users/leeovery/Code/tick/internal/task/notes.go:27-48 -- Custom MarshalJSON/UnmarshalJSON for Note with ISO 8601 timestamp formatting
  - /Users/leeovery/Code/tick/internal/task/notes.go:52-61 -- ValidateNoteText: checks non-empty after trimming, max 500 chars via utf8.RuneCountInString
  - /Users/leeovery/Code/tick/internal/task/notes.go:64-66 -- TrimNoteText helper
  - /Users/leeovery/Code/tick/internal/task/task.go:52 -- Notes []Note field with json:"notes,omitempty" on Task struct
  - /Users/leeovery/Code/tick/internal/task/task.go:70 -- Notes []Note on taskJSON intermediary struct for serialization
  - /Users/leeovery/Code/tick/internal/task/task.go:89,125 -- Notes passed through MarshalJSON/UnmarshalJSON on Task
- Notes: Implementation is correct and complete. The Note struct uses json:"-" on both fields to force custom serialization, consistent with how Task handles timestamps. The noteJSON intermediary struct provides the actual JSON keys ("text", "created"). ValidateNoteText uses utf8.RuneCountInString for proper Unicode character counting. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/task/notes_test.go:10-55 -- TestValidateNoteText: 6 subtests covering valid text, empty text, whitespace-only text, exactly 500 chars, 501 chars, and TrimNoteText
  - /Users/leeovery/Code/tick/internal/task/notes_test.go:57-108 -- TestNoteMarshalJSON: serialization to JSON with timestamp verification, deserialization from JSON with field and timestamp verification
  - /Users/leeovery/Code/tick/internal/task/notes_test.go:110-177 -- TestNoteTaskJSON: round-trip through Task JSON (2 notes preserved with correct text and timestamps), omitempty verification (nil Notes omitted from JSON output)
- Notes: All four specified edge cases are covered (empty, whitespace-only, 500 chars, 501 chars). JSON serialization tested at both Note level and Task level. The omitempty behavior is verified. Tests are focused and not redundant -- each subtest verifies a distinct behavior. Tests would fail if the feature broke.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run() subtests, "it does X" naming format, t.Helper() convention. Follows the same pattern as tags.go/refs.go for validation functions. Uses the project's FormatTimestamp/TimestampFormat constants for timestamp handling.
- SOLID principles: Good. Note type has single responsibility (data + serialization). Validation is separated into standalone functions (ValidateNoteText, TrimNoteText) reusable by callers. Clean separation between domain model and storage concerns.
- Complexity: Low. Straightforward validation logic, clean marshal/unmarshal with no branching complexity.
- Modern idioms: Yes. Uses utf8.RuneCountInString for character counting (not len()), json:"-" with custom marshaler pattern, proper error wrapping with fmt.Errorf and %w.
- Readability: Good. Self-documenting function names, doc comments on all exported types and functions, consistent with existing codebase style.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None. Implementation is clean, well-tested, and follows established patterns.
