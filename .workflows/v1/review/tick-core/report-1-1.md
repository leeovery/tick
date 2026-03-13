TASK: Task model & ID generation

ACCEPTANCE CRITERIA:
- [ ] Task struct has all 10 fields with correct Go types
- [ ] ID format matches `tick-{6 hex chars}` pattern
- [ ] IDs are generated using `crypto/rand`
- [ ] Collision retry works up to 5 times then errors
- [ ] Input IDs are normalized to lowercase
- [ ] Title validation enforces non-empty, max 500 chars, no newlines, trims whitespace
- [ ] Priority validation rejects values outside 0-4
- [ ] Self-references in `blocked_by` and `parent` are rejected
- [ ] Timestamps are ISO 8601 UTC

STATUS: Complete

SPEC CONTEXT: The spec defines a Task with 10 fields (id, title, status, priority, description, blocked_by, parent, created, updated, closed). IDs use `tick-{6 hex}` format via `crypto/rand`, case-insensitive with lowercase normalization. Collision retry up to 5 times. Title: required, non-empty, max 500 chars, no newlines, trim whitespace. Priority: integer 0-4, default 2. Status enum: open, in_progress, done, cancelled. Timestamps: ISO 8601 UTC (`YYYY-MM-DDTHH:MM:SSZ`). Self-references in blocked_by and parent rejected. Optional fields omitted from JSON when empty/null.

IMPLEMENTATION:
- Status: Implemented
- Location: `/Users/leeovery/Code/tick/internal/task/task.go` (lines 1-232)
- Notes:
  - Task struct (line 43-54): All 10 fields present with correct types -- `ID` (string), `Title` (string), `Status` (Status/string), `Priority` (int), `Description` (string), `BlockedBy` ([]string), `Parent` (string), `Created` (time.Time), `Updated` (time.Time), `Closed` (*time.Time). Optional fields use `omitempty` JSON tags.
  - Status enum (lines 18-27): Four constants defined -- StatusOpen, StatusInProgress, StatusDone, StatusCancelled with correct string values.
  - ID generation (lines 128-143): Uses `crypto/rand` for 3 random bytes, hex-encoded to 6 chars, prefixed with `tick-`. Loop retries up to `maxIDRetries` (5). Returns descriptive error on exhaustion.
  - NormalizeID (lines 146-148): Simple `strings.ToLower` -- correct.
  - ValidateTitle (lines 152-164): Trims whitespace internally, checks non-empty, checks for `\n` and `\r`, validates rune count <= 500. Uses `utf8.RuneCountInString` for proper Unicode handling.
  - TrimTitle (lines 167-169): Separate function for whitespace trimming.
  - ValidatePriority (lines 172-177): Rejects values outside 0-4 range.
  - ValidateBlockedBy (lines 180-187): Checks for self-reference using NormalizeID for case-insensitive comparison.
  - ValidateParent (lines 189-198): Checks for self-reference, allows empty parent.
  - FormatTimestamp (lines 201-203): Formats as ISO 8601 UTC using Go format string `"2006-01-02T15:04:05Z"`.
  - NewTask (lines 207-232): Creates task with trimmed title, validated, generates ID with collision check, sets defaults (priority 2, status open, timestamps to now UTC truncated to seconds).
  - JSON marshaling (lines 71-124): Custom MarshalJSON/UnmarshalJSON via taskJSON intermediary struct handles timestamp string conversion and optional field omission.

TESTS:
- Status: Adequate
- Coverage: All 13 specified test cases covered, plus additional edge cases
- Test file: `/Users/leeovery/Code/tick/internal/task/task_test.go` (lines 1-433)
- Specified tests matched:
  1. "it generates IDs matching tick-{6 hex} pattern" -- line 13
  2. "it retries on collision up to 5 times" -- line 25
  3. "it errors after 5 collision retries" -- line 45
  4. "it normalizes IDs to lowercase" -- line 59
  5. "it rejects empty title" -- line 69
  6. "it rejects title exceeding 500 characters" -- line 76
  7. "it rejects title with newlines" -- line 84
  8. "it trims whitespace from title" -- line 91
  9. "it rejects priority outside 0-4" -- line 131 (table-driven with -1, 5, 100)
  10. "it rejects self-reference in blocked_by" -- line 165
  11. "it rejects self-reference in parent" -- line 181
  12. "it sets default priority to 2 when not specified" -- line 204
  13. "it sets created and updated timestamps to current UTC time" -- line 220
- Additional tests (not over-tested, all valuable):
  - "it rejects whitespace-only title" -- edge case from spec (line 99)
  - "it accepts valid title at 500 characters" -- boundary test (line 106)
  - "it counts multi-byte Unicode characters as single characters" -- verifies rune-based counting (line 114)
  - "it accepts valid priorities" (0-4 each) -- confirms valid range (line 152)
  - "it accepts valid blocked_by references" -- happy path (line 172)
  - "it accepts valid parent reference" / "it accepts empty parent" -- happy paths (lines 188, 195)
  - "it has all 10 fields with correct Go types" -- structural verification (line 243)
  - "it defines all four status constants" -- enum verification (line 281)
  - "it formats timestamps as ISO 8601 UTC" -- timestamp format (line 298)
  - JSON round-trip tests (minimal, full, format verification, optional field omission) -- lines 308-433
- Notes: Tests use subtests with descriptive names. Table-driven where appropriate (priority validation). Edge cases well covered. No redundant or bloated tests -- each verifies a distinct behavior.

CODE QUALITY:
- Project conventions: Followed. Uses `internal/` package structure per Go convention. Table-driven tests with subtests. Exported functions documented. Error handling explicit with descriptive messages.
- SOLID principles: Good. Single responsibility -- task.go handles only task model, ID generation, and field validation. Functions are small and focused. Dependency inversion via `exists` callback for ID collision checking.
- Complexity: Low. All functions have simple, linear control flow. GenerateID has one loop with early return. ValidateTitle has 3 sequential checks. No complex branching.
- Modern idioms: Yes. Uses `crypto/rand` (not `math/rand`). `utf8.RuneCountInString` for Unicode-aware length. Custom JSON marshaling via intermediate struct. `errors.New` for simple errors, `fmt.Errorf` for formatted errors.
- Readability: Good. Constants are named and grouped logically. Functions are self-documenting. The taskJSON intermediary struct cleanly separates serialization concerns from the domain model.
- Security: No concerns. Uses cryptographic random for ID generation as spec requires.
- Performance: No concerns. Simple operations, no unnecessary allocations or loops.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- Error message casing: Spec says "Failed to generate unique ID after 5 attempts..." (capital F), implementation uses "failed to..." (lowercase f). The lowercase form follows Go convention for error strings (per Go Code Review Comments: "Error strings should not be capitalized"). This is a deliberate and well-reasoned deviation documented in analysis files. Acceptable.
- ValidateTitle internally trims before validating, but TrimTitle is also exported separately. NewTask calls both TrimTitle then ValidateTitle, causing a double-trim. Not a bug (trimming is idempotent), but the API design could be slightly cleaner. Very minor.
