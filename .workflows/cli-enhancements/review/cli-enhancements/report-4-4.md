TASK: Extract generic deduplication and validation helpers in internal/task (cli-enhancements-5-2)

ACCEPTANCE CRITERIA:
- DeduplicateTags and DeduplicateRefs delegate to a shared helper
- No behavioral change -- all existing tests pass without modification
- The shared helper is unexported (internal implementation detail)

STATUS: Complete

SPEC CONTEXT: Tags and refs both require deduplication of string slices with normalization (tags: trim+lowercase, refs: trim). The specification defines both as []string fields with silent deduplication on input. The analysis (cycle 1) identified that DeduplicateTags and DeduplicateRefs implemented identical algorithms differing only in the normalizer function, and recommended extracting a shared `deduplicateStrings` helper. The optional extraction of a `validateCollection` helper was explicitly marked as optional and later confirmed as borderline by the cycle 3 analysis.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/task/helpers.go:1-19` -- New file containing `deduplicateStrings(items []string, normalize func(string) string) []string`, an unexported generic helper that normalizes each item, filters empties, and returns unique items in first-occurrence order using a seen-map.
  - `/Users/leeovery/Code/tick/internal/task/tags.go:38-39` -- `DeduplicateTags` is now a one-line wrapper: `return deduplicateStrings(tags, NormalizeTag)`
  - `/Users/leeovery/Code/tick/internal/task/refs.go:38-39` -- `DeduplicateRefs` is now a one-line wrapper: `return deduplicateStrings(refs, strings.TrimSpace)`
- Notes: All three acceptance criteria are met. The shared helper is unexported (lowercase `deduplicateStrings`). Both `DeduplicateTags` and `DeduplicateRefs` delegate to it. The optional `validateCollection` helper was not extracted, which is acceptable per the task description ("Optionally: extract a validateCollection helper") and the cycle 3 analysis confirmation that it was borderline with only 2 instances.

TESTS:
- Status: Adequate
- Coverage: No new dedicated test file for `helpers.go`. The shared helper is fully exercised through existing tests:
  - `/Users/leeovery/Code/tick/internal/task/tags_test.go:195-234` -- `TestDeduplicateTags` with subtests for dedup preserving order, filtering empties, and normalizing during dedup
  - `/Users/leeovery/Code/tick/internal/task/refs_test.go:76-81` -- Dedup tests within `TestValidateRefs`
  - `/Users/leeovery/Code/tick/internal/task/refs_test.go:156-163` -- Dedup tests within `TestParseRefs`
  - `/Users/leeovery/Code/tick/internal/task/tags_test.go:237-282` -- `TestValidateTags` exercising dedup + validation integration
  - `/Users/leeovery/Code/tick/internal/task/refs_test.go:75-121` -- `TestValidateRefs` exercising dedup + validation integration
- Notes: This is a pure refactoring task. The acceptance criteria explicitly state "All existing tag and ref deduplication tests pass unchanged" and "All existing tag and ref validation tests pass unchanged." No new tests are required since behavior is preserved. The existing tests cover: first-occurrence ordering, empty string filtering, normalization during dedup, and dedup-before-validation (11 items deduped to 10). If the shared helper broke, multiple existing tests would fail. Test coverage is appropriate for a refactoring task.

CODE QUALITY:
- Project conventions: Followed. Uses Go stdlib only (no external dependencies). File naming follows project pattern (helpers.go for shared utilities). Unexported function for internal-only usage.
- SOLID principles: Good. The helper embodies the Open/Closed principle -- new slice-type fields can reuse `deduplicateStrings` by providing a normalizer function, without modifying the helper. Single responsibility -- helpers.go contains only generic utility functions.
- Complexity: Low. The `deduplicateStrings` function is a straightforward 14-line implementation with O(n) time complexity using a seen-map. No nested loops, no complex branching.
- Modern idioms: Yes. Uses functional parameter (normalizer callback) for customization. Idiomatic Go map-based dedup pattern.
- Readability: Good. Clear function name, descriptive godoc comment, and the normalize callback makes the customization point obvious. Both wrappers (`DeduplicateTags`, `DeduplicateRefs`) are self-explanatory one-liners.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The parallel structure of `ValidateTags` and `ValidateRefs` (both ~20 lines with identical flow: early-return on empty, call dedup, validate each, check max count) remains duplicated. The optional `validateCollection` helper was deliberately not extracted. This is acceptable given the cycle 3 analysis confirmed it as borderline with only 2 instances, but it would be worth revisiting if a third slice-type field is added.
