TASK: Add Transition struct and Transitions field to Task (acps-2-1)

ACCEPTANCE CRITERIA:
- Task.Transitions array field serializes/deserializes correctly in JSONL with `auto` boolean
- Empty transitions array omitted from JSON
- Backward-compatible deserialization of tasks without transitions field
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: The spec defines a `transitions` array on each task recording `from`, `to`, `at` (timestamp), and `auto` (boolean for cascade-triggered transitions). Stored in JSONL as part of the Task struct. Growth is bounded since tasks don't transition many times.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/task/transition_history.go:12-17` -- TransitionRecord struct with From, To, At, Auto fields
  - `/Users/leeovery/Code/tick/internal/task/transition_history.go:19-25` -- transitionRecordJSON serialization helper
  - `/Users/leeovery/Code/tick/internal/task/transition_history.go:28-54` -- MarshalJSON/UnmarshalJSON with ISO 8601 timestamp handling
  - `/Users/leeovery/Code/tick/internal/task/task.go:53` -- Transitions field on Task struct with `json:"transitions,omitempty"`
  - `/Users/leeovery/Code/tick/internal/task/task.go:72` -- Transitions field on taskJSON helper struct
  - `/Users/leeovery/Code/tick/internal/task/task.go:92,129` -- Transitions wired into Task MarshalJSON/UnmarshalJSON
- Notes: Implementation follows the same pattern as Notes (custom JSON marshal/unmarshal with dedicated helper struct). The `omitempty` tag ensures empty/nil transitions arrays are omitted from JSON output. The struct fields use `json:"-"` tags with custom marshal/unmarshal, consistent with how Task itself handles timestamps.

TESTS:
- Status: Adequate
- Coverage:
  - TransitionRecord marshal to JSON with ISO 8601 timestamp verified
  - TransitionRecord unmarshal from valid JSON verified
  - Round-trip marshal/unmarshal verified
  - Auto boolean false vs true preservation verified
  - Empty transitions omitted from task JSON verified
  - Task with transitions included in JSON and round-trips correctly verified
  - Backward-compatible deserialization of task JSON without transitions field verified (returns nil)
- Notes: All edge cases from the task description are covered. Tests are focused and not redundant -- each subtest verifies a distinct behavior. The auto boolean preservation test is particularly good, ensuring both false and true values survive serialization (important since false is the zero value for bool).

CODE QUALITY:
- Project conventions: Followed -- uses stdlib testing only, t.Run subtests, "it does X" naming, follows same pattern as Notes/Note for custom JSON serialization
- SOLID principles: Good -- TransitionRecord has single responsibility (data + serialization), separated from Task struct
- Complexity: Low -- straightforward marshal/unmarshal with no branching complexity
- Modern idioms: Yes -- custom JSON marshal/unmarshal is idiomatic Go
- Readability: Good -- clear naming, documented exported types and methods
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
