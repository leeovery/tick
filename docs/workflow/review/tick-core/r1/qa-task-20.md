TASK: JSON formatter -- list, show, stats output (tick-core-4-4)

ACCEPTANCE CRITERIA:
- [ ] Implements full Formatter interface
- [ ] Empty list -> `[]`
- [ ] blocked_by/children always `[]` when empty
- [ ] parent/closed omitted when null
- [ ] description always present
- [ ] snake_case keys throughout
- [ ] Stats nested with 5 priority entries
- [ ] All output valid JSON
- [ ] 2-space indented

STATUS: Complete

SPEC CONTEXT:
The specification (section "JSON Format") states: "Available via `--json` flag for compatibility and debugging. Standard JSON output." The JSON formatter is the third formatter alongside TOON and Pretty, selected when `--json` flag is passed or via `NewFormatter(FormatJSON)`. The spec defines the task schema with 10 fields, where `blocked_by` defaults to `[]`, `parent` defaults to `null`, `description` defaults to `""`, and `closed` defaults to `null`. Optional fields are "omitted when empty/null (not serialized as null)" in JSONL storage. The JSON output format should use snake_case to match the JSONL storage convention.

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/internal/cli/json_formatter.go:1-211`
- Notes:
  - `JSONFormatter` struct implements all 6 methods of the `Formatter` interface (line 14: compile-time verification via `var _ Formatter = (*JSONFormatter)(nil)`)
  - `FormatTaskList` (line 26-37): Uses `make([]jsonTaskListItem, 0, len(tasks))` which correctly produces `[]` for both nil and empty input slices, avoiding Go's nil-slice-to-null JSON gotcha
  - `FormatTaskDetail` (line 67-90): Uses dedicated `jsonTaskDetail` struct with `omitempty` on `Parent` and `Closed` fields (lines 56, 59); `Description` has no `omitempty` so always serialized; `BlockedBy` and `Children` use `toJSONRelated()` which returns initialized empty slices
  - `toJSONRelated` (line 94-100): `make([]jsonRelatedTask, 0, len(related))` ensures non-nil even when input is nil
  - `FormatStats` (line 165-190): Produces nested object with `total`, `by_status`, `workflow`, `by_priority`; always creates 5 priority entries via loop (line 166-171)
  - `FormatTransition` (line 110-116): Returns `{"id", "from", "to"}` as specified
  - `FormatDepChange` (line 126-132): Returns `{"action", "task_id", "blocked_by"}` as specified
  - `FormatMessage` (line 198-200): Returns `{"message"}` as specified
  - `marshalIndentJSON` (line 204-210): Uses `json.MarshalIndent(v, "", "  ")` for 2-space indentation
  - All JSON struct tags use snake_case consistently
  - Wired into `NewFormatter` at `/Users/leeovery/Code/tick/internal/cli/format.go:178-179`

TESTS:
- Status: Adequate
- Coverage: `/Users/leeovery/Code/tick/internal/cli/json_formatter_test.go:1-570`
  - "it formats list as JSON array" (line 16) -- verifies array structure and field values
  - "it formats empty list as [] not null" (line 53) -- tests both empty slice AND nil slice input
  - "it formats show with all fields" (line 69) -- verifies all fields including blocked_by/children arrays with items
  - "it omits parent/closed when null" (line 157) -- verifies omission via map key existence check
  - "it includes blocked_by/children as empty arrays" (line 190) -- empty slice input
  - "it includes blocked_by/children as empty arrays even with nil input" (line 231) -- nil slice input (Go nil-to-null edge case)
  - "it formats description as empty string not null" (line 270) -- verifies presence and empty string value
  - "it uses snake_case for all keys" (line 307) -- checks all expected keys present AND no camelCase keys
  - "it formats stats as structured nested object" (line 354) -- verifies nested structure and all values
  - "it includes 5 priority rows even at zero" (line 428) -- all-zero stats, verifies 5 entries
  - "it formats transition/dep/message as JSON objects" (line 459) -- covers transition, dep add, dep remove, message
  - "it produces valid parseable JSON" (line 515) -- validates all output types via `json.Valid()`
  - "it uses 2-space indentation" (line 556) -- string contains check for 2-space indent
- Notes: All 11 tests from the task plan are present plus 2 additional tests (nil input for blocked_by/children, and 2-space indentation). The nil input test is a valuable edge case not redundant with the empty-slice test. The 2-space indentation test directly verifies an acceptance criterion. Tests verify behavior (parsed JSON output) rather than implementation details. No over-testing observed.

CODE QUALITY:
- Project conventions: Followed. Compile-time interface check present. Exported types/functions documented with comments. File structure follows existing patterns (see `toon_formatter.go`, `pretty_formatter.go`).
- SOLID principles: Good. Single responsibility -- JSONFormatter only handles JSON rendering. Open/closed -- implements the Formatter interface without modifying it. Interface segregation -- Formatter interface is cohesive with 6 related methods.
- Complexity: Low. Each method is straightforward struct construction + marshal. No branching logic except the nil check for `Closed` timestamp.
- Modern idioms: Yes. Uses struct tags for JSON key naming. Uses `omitempty` appropriately. Compile-time interface verification.
- Readability: Good. Dedicated structs for each output type make the JSON shape explicit and self-documenting. Helper function `marshalIndentJSON` eliminates repetition. `toJSONRelated` clearly handles the nil-slice edge case.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `marshalIndentJSON` function returns "null" on marshal failure (line 207-208). This is an acceptable fallback since the controlled struct types should never fail marshaling, but a log warning could aid debugging in the unlikely event it occurs.
- The `jsonRelatedTask` struct (line 40-44) has the same field layout as `RelatedTask` (format.go:87-91), and the conversion at line 97 uses a direct type conversion `jsonRelatedTask(r)`. This works because the fields are identical in name, type, and order. If either struct changes independently, this will break at compile time, which is a safe failure mode.
