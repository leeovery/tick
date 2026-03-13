TASK: cli-enhancements-2-1 — Add Type field to Task model and JSONL serialization

ACCEPTANCE CRITERIA:
- Task struct has Type string field with json:"type,omitempty"
- JSONL serialization (MarshalJSON/UnmarshalJSON) includes Type
- Validation: closed set (bug, feature, task, chore), case-insensitive, trimmed, stored lowercase
- ValidateTypeNotEmpty for --type flag context (empty -> error mentioning --clear-type)
- Edge cases: empty string on --type, mixed-case input, invalid type value, whitespace-only input

STATUS: Complete

SPEC CONTEXT: Task Types - string field classifying work kind. Allowed values: bug, feature, task, chore (closed set). Case-insensitive input, trimmed, stored lowercase. JSONL: string field with omitempty. SQLite column and CLI flags are separate tasks (2-2 through 2-6).

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/task/task.go:48 — Type field on Task struct with `json:"type,omitempty"`
  - /Users/leeovery/Code/tick/internal/task/task.go:66 — Type field on taskJSON struct with `json:"type,omitempty"`
  - /Users/leeovery/Code/tick/internal/task/task.go:86 — Type included in MarshalJSON
  - /Users/leeovery/Code/tick/internal/task/task.go:121 — Type included in UnmarshalJSON
  - /Users/leeovery/Code/tick/internal/task/task.go:230-244 — allowedTypes var and ValidateType function
  - /Users/leeovery/Code/tick/internal/task/task.go:246-249 — NormalizeType (TrimSpace + ToLower)
  - /Users/leeovery/Code/tick/internal/task/task.go:251-258 — ValidateTypeNotEmpty with --clear-type hint
- Notes: Implementation is clean and complete. ValidateType accepts empty string (optional field), while ValidateTypeNotEmpty is used at the CLI layer when --type flag is explicitly provided. NormalizeType correctly composes TrimSpace and ToLower. The allowedTypes slice is unexported (good encapsulation). All four allowed values match the spec exactly.

TESTS:
- Status: Adequate
- Coverage:
  - TestValidateType: all 4 valid types accepted, invalid type rejected with helpful error listing allowed values, empty string allowed for optional field, mixed-case invalid type after normalization
  - TestNormalizeType: table-driven tests covering whitespace+case combos ("  BUG  " -> "bug", "Feature" -> "feature", " TASK " -> "task", "  chore  " -> "chore", "" -> "", "  " -> "")
  - TestValidateTypeNotEmpty: empty string rejected with --clear-type mention, whitespace-only rejected after normalization with --clear-type mention
  - TestTaskMarshalJSON subtests: marshals type when set (line 487), omits type when empty (line 517), unmarshals type from JSON (line 539), backward compat without type field (line 550), omitempty check includes "type" (line 480)
- Notes: All four edge cases from the task definition are covered: empty string (ValidateTypeNotEmpty), mixed-case input (NormalizeType table + ValidateType mixed-case subtest), invalid type value (ValidateType "enhancement"), whitespace-only (NormalizeType "  " + ValidateTypeNotEmpty whitespace). Tests are focused and not redundant. Would fail if the feature broke.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, "it does X" naming, fmt.Errorf with %w wrapping, same file organization pattern as other validators.
- SOLID principles: Good. ValidateType, NormalizeType, and ValidateTypeNotEmpty each have a single responsibility. NormalizeType is a pure function. Validation is separated from normalization (caller normalizes first, then validates).
- Complexity: Low. Each function is trivial: NormalizeType is a one-liner, ValidateType is a simple loop, ValidateTypeNotEmpty is a simple check.
- Modern idioms: Yes. Idiomatic Go: unexported var for closed set, separate normalize/validate pattern matching existing codebase conventions (TrimTitle/ValidateTitle, TrimDescription/ValidateDescriptionUpdate).
- Readability: Good. Function names are self-documenting. Doc comments on all exported functions. Error messages are clear and actionable (mentioning --clear-type when appropriate).
- Issues: None

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- (none)
