TASK: cli-enhancements-3-1 -- Tags field on Task model with validation and JSONL serialization

ACCEPTANCE CRITERIA:
- Task struct has Tags []string field with json:"tags,omitempty"
- Tag validation enforces kebab-case regex [a-z0-9]+(-[a-z0-9]+)*, max 30 chars per tag, max 10 tags after dedup
- Input trimmed, lowercased, silently deduplicated before validation

STATUS: Complete

SPEC CONTEXT: Tags are a []string field for user-defined labels. Strict kebab-case enforced. No spaces, commas, leading/trailing hyphens, double hyphens. Input trimmed and lowercased. Max 30 chars per tag, max 10 tags per task after dedup. Silent deduplication on input. JSONL: JSON array with omitempty. Spec explicitly lists the regex pattern [a-z0-9]+(-[a-z0-9]+)*.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/task/task.go:49 -- Tags []string field with json:"tags,omitempty"
  - /Users/leeovery/Code/tick/internal/task/task.go:66 -- Tags in taskJSON mirror struct for serialization
  - /Users/leeovery/Code/tick/internal/task/task.go:86-87 -- Tags marshaled in MarshalJSON
  - /Users/leeovery/Code/tick/internal/task/task.go:124 -- Tags unmarshaled in UnmarshalJSON
  - /Users/leeovery/Code/tick/internal/task/tags.go:1-61 -- Full validation module (NormalizeTag, ValidateTag, DeduplicateTags, ValidateTags)
  - /Users/leeovery/Code/tick/internal/task/helpers.go:5-19 -- Shared deduplicateStrings helper
- Notes:
  - Struct tag correctly uses json:"tags,omitempty" as specified
  - Regex pattern ^[a-z0-9]+(-[a-z0-9]+)*$ matches spec exactly
  - Constants: maxTagLength=30, maxTagsPerTask=10 match spec
  - NormalizeTag correctly trims whitespace and lowercases
  - DeduplicateTags delegates to shared deduplicateStrings helper (DRY pattern from Phase 5 refactoring)
  - ValidateTags correctly: (1) early-returns on empty, (2) deduplicates first, (3) validates each, (4) checks count
  - ValidateTag length check uses len() (byte count) which is correct since regex only allows ASCII chars
  - Validation order in ValidateTag: empty check -> length check -> regex match. This is correct -- rejects empty and too-long tags before regex, and the regex only passes valid kebab-case

TESTS:
- Status: Adequate
- Coverage:
  - TestTagMarshalJSON (3 subtests): marshal with tags, omitempty for nil/empty tags, unmarshal from JSON -- covers JSONL serialization roundtrip
  - TestValidateTag (9 subtests): valid single-segment, valid multi-segment, double hyphens, leading hyphen, trailing hyphen, spaces, 31-char rejection, 30-char acceptance, empty tag, additional formats (v2, a1-b2-c3)
  - TestNormalizeTag: table-driven with whitespace trimming, uppercase, mixed case, already lowercase, empty, whitespace-only
  - TestDeduplicateTags (3 subtests): dedup preserving order, empty string filtering, normalization during dedup
  - TestValidateTags (5 subtests): 11 tags deduped to 10 (passes), 11 unique tags (rejects), empty strings in list, invalid tag in list caught, empty/nil list accepted
- All 7 edge cases from the task specification are covered:
  1. double hyphens -- TestValidateTag line 108
  2. leading/trailing hyphens -- TestValidateTag lines 115, 122
  3. mixed-case normalization -- TestNormalizeTag line 171, TestDeduplicateTags line 224
  4. 31-char tag -- TestValidateTag line 136
  5. 11 tags deduped to 10 -- TestValidateTags line 238
  6. empty string in list -- TestDeduplicateTags line 210, TestValidateTags line 257
  7. tag with spaces -- TestValidateTag line 129
- Tests would fail if the feature broke (regex change, constant change, normalization removal)
- No over-testing; each test covers a distinct behavior

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing only, t.Run subtests, "it does X" naming, no testify, error wrapping with fmt.Errorf. File organization (separate tags.go file) follows the pattern of other field modules (refs.go).
- SOLID principles: Good. Single responsibility -- tags.go handles only tag validation/normalization. DeduplicateTags delegates to shared deduplicateStrings helper (open/closed principle -- new field types can reuse). ValidateTags composes ValidateTag per element.
- Complexity: Low. Each function is linear, no nesting beyond simple if/for. Cyclomatic complexity is minimal.
- Modern idioms: Yes. Compiled regex var, proper use of strings package, idiomatic Go error returns.
- Readability: Good. Function names are self-documenting (NormalizeTag, ValidateTag, DeduplicateTags, ValidateTags). Constants have clear names. Comments on exported functions.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
