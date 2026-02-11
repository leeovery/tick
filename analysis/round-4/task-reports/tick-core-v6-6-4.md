# Task tick-core-6-4: Consolidate Formatter Duplication and Fix Unicode Arrow (V6 Only -- Analysis Phase 6)

## Note
This is an analysis refinement task that only exists in V6. Standalone quality assessment.

## Task Summary
This task addresses two related issues: (1) `FormatTransition` and `FormatDepChange` methods were duplicated identically across `ToonFormatter` and `PrettyFormatter`, and (2) transition output used ASCII `->` instead of the spec-mandated Unicode right arrow (U+2192). The solution extracts a `baseFormatter` embedded struct providing both shared methods, embeds it in both formatters, and replaces the ASCII arrow with the Unicode character.

## V6 Implementation

### Architecture & Design
The implementation introduces a `baseFormatter` struct in `internal/cli/format.go` that provides `FormatTransition` and `FormatDepChange` as methods. Both `ToonFormatter` and `PrettyFormatter` embed `baseFormatter`, inheriting these methods via Go's struct embedding (composition). This is textbook Go composition -- small, focused embedded types providing shared behavior without inheritance.

Key design decisions:
- **Placement in `format.go`**: Correct location. The `baseFormatter` lives alongside the `Formatter` interface and `StubFormatter`, keeping the formatting contract and shared implementation co-located.
- **Not embedding in `JSONFormatter`**: The developer correctly evaluated step 5 of the task plan ("Check JsonFormatter -- if it also has identical implementations, embed there too") and determined that `JSONFormatter` has structurally different implementations (it outputs structured JSON objects with `id`, `from`, `to` fields rather than plain text strings). This is the right call.
- **Unicode consistency**: The codebase now uniformly uses `\u2192` for arrows in both `dependency.go` cycle errors and formatter transition output.

The net result is four methods removed (two from each formatter), two methods added to `baseFormatter` -- a clean consolidation.

### Code Quality
The implementation is clean and idiomatic:

- **Exported type documentation**: `baseFormatter` has a clear doc comment explaining its purpose and which formatters embed it.
- **Method documentation**: Both `FormatTransition` and `FormatDepChange` have doc comments on the `baseFormatter` methods.
- **Interface comment updated**: The `FormatTransition` comment in the `Formatter` interface was updated from `"open -> in_progress"` to `"open \u2192 in_progress"` to match the new behavior.
- **Minimal diff**: Only the necessary files were touched. The change is surgical -- no unrelated modifications.
- **`FormatDepChange` handles action branching via string comparison**: The `if action == "removed"` pattern is preserved from the original. This is adequate for two cases but could theoretically be fragile if more actions were added. However, this is an existing pattern, not introduced by this task.

One minor observation: the `baseFormatter` is unexported (lowercase), which is correct since it is an internal implementation detail not meant for external consumption.

### Test Coverage
Test coverage is thorough and well-structured:

**New test file `base_formatter_test.go` (96 lines):**
1. **`TestBaseFormatter`** -- Tests `baseFormatter` directly:
   - Verifies `FormatTransition` contains Unicode right arrow (basic assertion)
   - Table-driven test with 3 cases (open->in_progress, in_progress->done, done->open) verifying spec format `id: old_status \u2192 new_status`
   - `FormatDepChange` add case
   - `FormatDepChange` remove case

2. **`TestAllFormattersProduceConsistentTransitionOutput`** -- Cross-formatter consistency test:
   - Instantiates both `ToonFormatter` and `PrettyFormatter`
   - Verifies both produce identical transition output
   - Three-way assertion: toon matches expected, pretty matches expected, toon matches pretty

**Updated existing tests:**
- `pretty_formatter_test.go`: Updated expected string from `->` to `\u2192`
- `toon_formatter_test.go`: Updated expected string from `->` to `\u2192`
- `transition_test.go`: Updated expected output from `->` to `\u2192`
- `format_integration_test.go`: Updated both toon and pretty transition assertions to use `\u2192`

All four acceptance criteria from the task plan are covered:
1. FormatTransition/FormatDepChange exist in one place only -- verified by the consistency test
2. Unicode arrow -- directly asserted in multiple tests
3. All formatters produce correct output -- existing formatter tests plus new consistency test
4. All existing formatter tests pass -- assertions updated

### Spec Compliance
- **Unicode arrow (spec line 639)**: Correctly uses U+2192 (`\u2192`) in transition output
- **Internal consistency**: Now matches `dependency.go` cycle error messages which already used `\u2192`
- **Format**: Output format `tick-id: old_status \u2192 new_status` matches the spec

### golang-pro Skill Compliance
- **Table-driven tests with subtests**: The `FormatTransition matches spec format` test uses table-driven pattern with `t.Run` subtests -- fully compliant
- **All exported functions documented**: `baseFormatter`, `FormatTransition`, `FormatDepChange` all have doc comments
- **Error handling**: Not applicable (no error paths in this change)
- **No panics for error handling**: N/A
- **No ignored errors**: N/A
- **Interfaces**: Uses Go embedding/composition correctly rather than inheritance
- **gofmt compliance**: Code appears properly formatted

## Quality Assessment

### Strengths
1. **Clean application of Go composition**: The `baseFormatter` embedded struct is the idiomatic Go way to share method implementations across types that implement the same interface. No inheritance, no code generation, just embedding.
2. **Thorough test updates**: Every test asserting on the old ASCII arrow was found and updated. The new `base_formatter_test.go` covers the shared implementation directly and adds a cross-formatter consistency test that will catch future regressions if someone overrides the embedded method.
3. **Correct evaluation of JSONFormatter**: The developer correctly decided not to embed `baseFormatter` in `JSONFormatter` since its implementations are structurally different (JSON objects vs plain text strings).
4. **Minimal, focused diff**: 130 insertions, 38 deletions across 10 files. No scope creep, no unnecessary changes.
5. **Interface documentation updated**: The `Formatter` interface comment was updated to reflect the new Unicode arrow, keeping docs accurate.

### Weaknesses
1. **No negative/edge case tests for `FormatDepChange`**: The tests only cover `"added"` and `"removed"` actions. There is no test for an unexpected action string (e.g., what happens with `"modified"`?). The current implementation would fall through to the `"added"` format string, which could be surprising. A minor issue since this is inherited behavior, not introduced.
2. **`JSONFormatter` not included in the consistency test**: `TestAllFormattersProduceConsistentTransitionOutput` only tests `ToonFormatter` and `PrettyFormatter`. While JSON output is structurally different (and thus not expected to match), the test name implies "all formatters" but only covers two of three. This is arguably correct since JSON has different output semantics, but the test name could be more precise (e.g., `TestTextFormattersProduceConsistentTransitionOutput`).
3. **No `require`/`testify` usage**: Tests use raw `t.Errorf` throughout. This is standard library only (which is fine and even preferred by some Go teams), but deeper assertion failures in table-driven tests will report all failures rather than failing fast. This is a project-wide pattern, not specific to this task.

### Overall Quality Rating
Excellent

The task is a clean, well-scoped refactoring that eliminates real code duplication and fixes a spec compliance issue. The implementation uses idiomatic Go composition, the test coverage is thorough with both unit and cross-formatter consistency tests, and every existing test was correctly updated. The diff is minimal and focused with no regressions introduced.
