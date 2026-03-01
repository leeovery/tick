TASK: cli-enhancements-4-1 -- Refs field on Task model with validation and JSONL serialization

ACCEPTANCE CRITERIA:
- Task struct has Refs []string field with json:"refs,omitempty"
- Ref validation: non-empty, no commas, no whitespace, max 200 chars, max 10 per task, silent dedup

STATUS: Complete

SPEC CONTEXT:
External references are a []string field for cross-system links (gh-123, JIRA-456, URLs). Validation requires non-empty, no commas, no whitespace (contiguous strings only), input trimmed before validation, max 200 chars per ref, max 10 refs per task (validated after deduplication), no format validation, silent dedup. JSONL uses JSON array with omitempty.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/task/task.go:50 -- Refs []string `json:"refs,omitempty"` on Task struct
  - /Users/leeovery/Code/tick/internal/task/task.go:68 -- Refs []string `json:"refs,omitempty"` on taskJSON struct
  - /Users/leeovery/Code/tick/internal/task/task.go:87 -- Refs mapped in MarshalJSON
  - /Users/leeovery/Code/tick/internal/task/task.go:123 -- Refs mapped in UnmarshalJSON
  - /Users/leeovery/Code/tick/internal/task/refs.go:17-34 -- ValidateRef (single ref validation)
  - /Users/leeovery/Code/tick/internal/task/refs.go:38-40 -- DeduplicateRefs (delegates to shared deduplicateStrings)
  - /Users/leeovery/Code/tick/internal/task/refs.go:43-61 -- ValidateRefs (dedup + validate each + count check)
  - /Users/leeovery/Code/tick/internal/task/refs.go:65-78 -- ParseRefs (comma-split + dedup + validate)
  - /Users/leeovery/Code/tick/internal/task/helpers.go:5-19 -- Shared deduplicateStrings helper
- Notes:
  - ValidateRef uses len() (byte count) at line 30 rather than utf8.RuneCountInString() (rune count) for max length check. Since refs are constrained to have no whitespace and no commas (contiguous ASCII-like strings), this is unlikely to cause issues in practice, but is inconsistent with ValidateTitle which uses utf8.RuneCountInString(). Non-blocking.
  - ParseRefs calls DeduplicateRefs then ValidateRefs, but ValidateRefs internally calls DeduplicateRefs again -- double dedup. Already tracked as Phase 6 task cli-enhancements-6-3. Non-blocking.

TESTS:
- Status: Adequate
- Coverage:
  - TestValidateRef (7 subtests): comma rejection, whitespace rejection, 200-char acceptance, 201-char rejection, empty ref rejection, whitespace-only rejection, trimming, valid format acceptance (gh-123, JIRA-456, URL)
  - TestValidateRefs (5 subtests): silent dedup, 11 unique refs rejected, 11 refs deduped to 10 accepted, individual ref validation within list, empty/nil list acceptance
  - TestParseRefs (5 subtests): comma splitting, whitespace trimming, dedup, ref-with-whitespace rejection, empty input rejection
  - TestRefMarshalJSON (4 subtests): round-trip, omitempty on nil refs, unmarshal with refs, backward compat without refs field
- Notes:
  - All edge cases from the task description are covered: ref containing commas, ref containing whitespace, ref exactly 200 chars, 201-char ref, 11 refs deduped to 10, 11 unique refs, empty string in list (handled by deduplicateStrings filtering empties), whitespace-only ref
  - Tests are focused and non-redundant; each subtest verifies a distinct behavior
  - JSONL serialization round-trip and backward compatibility are both tested

CODE QUALITY:
- Project conventions: Followed. Uses t.Run subtests, t.Helper not needed (no test helpers), stdlib testing only, error wrapping with fmt.Errorf, functional composition (ValidateRefs calls ValidateRef per element)
- SOLID principles: Good. Single responsibility -- refs.go handles only ref validation/normalization. DeduplicateRefs delegates to shared deduplicateStrings (open/closed -- new field types can reuse). ValidateRefs composes ValidateRef per element.
- Complexity: Low. Each function is straightforward with clear control flow.
- Modern idioms: Yes. Uses unicode.IsSpace for whitespace detection (correct for all Unicode whitespace), strings.ContainsRune for comma check.
- Readability: Good. Well-named functions, clear documentation comments, consistent with the tags.go pattern.
- Issues: None blocking.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- ValidateRef uses len() (byte count) for max length check at refs.go:30. ValidateTitle uses utf8.RuneCountInString() (rune count). For practical purposes this is fine since refs are ASCII-like contiguous strings, but the inconsistency could be noted for future alignment.
- Double deduplication in ParseRefs -> ValidateRefs path is already tracked as Phase 6 task cli-enhancements-6-3.
