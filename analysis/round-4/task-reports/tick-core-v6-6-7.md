# tick-core-6-7: Add Explanatory Second Line to Child-Blocked-By-Parent Error

**Commit**: `f51c7b7`
**V6-only task** (no V5 equivalent)

## Task Summary

The task adds a second explanatory line to the child-blocked-by-parent dependency validation error message, aligning it with the spec (lines 405-408). The spec defines the error as a two-line message where the second line explains the rationale: `(would create unworkable task due to leaf-only ready rule)`. Prior to this commit, only the first line was emitted.

## V6 Implementation

### Architecture

Minimal and well-scoped. The change touches exactly two files in `internal/task/`:

- **`dependency.go`** (line 36): Appends `\n(would create unworkable task due to leaf-only ready rule)` to the existing `fmt.Errorf` format string in `validateChildBlockedByParent`.
- **`dependency_test.go`** (line 86): Updates the single assertion that checks the exact error text.

No structural or architectural changes. The error message lives in the correct validation layer (`internal/task`) and the "Error: " prefix is correctly added by the CLI dispatcher in `internal/cli/app.go:32`, maintaining proper separation of concerns.

### Code Quality

**Strengths**:

- The change is a single-line format string modification -- minimal surface area, minimal risk.
- The `\n` separator between the two lines is clean and straightforward.
- The function's doc comment already explains the rationale ("creates a deadlock with the leaf-only ready rule"), so the error message and the code documentation are aligned.

**Capitalization note**: The task plan explicitly called for fixing capitalization to "Cannot" (capital C) to match the spec. The implementation retains lowercase "cannot". This is actually the **correct Go convention** -- `go vet` and the Go community standard dictate that error strings should not start with a capital letter or end with punctuation (per [Effective Go](https://go.dev/doc/effective_go#errors) and `go vet` checks). Since the CLI layer prepends `"Error: "` via `fmt.Fprintf(a.Stderr, "Error: %s\n", err)`, the final user-visible output renders as `Error: cannot add dependency...` which is acceptable. The task plan's capitalization requirement conflicted with Go idiom, and the implementation correctly prioritized Go convention.

**Alignment note**: The spec shows the second line indented/aligned under the first. The implementation uses a bare `\n(` with no leading whitespace on the second line. Whether the CLI adds alignment padding is not evident, but the raw error string does not match the spec's visual alignment exactly. This is a minor cosmetic divergence.

### Test Coverage

The existing test `"it rejects child blocked by own parent"` (line 75) is updated to assert the full two-line error message including the rationale. The assertion uses exact string matching (`err.Error() != expected`), which is thorough.

**What is tested**:
- The complete two-line error message text
- Both the task ID and parent ID are correctly interpolated
- The `\n` separator and second line content

**What is NOT tested**:
- The task plan's acceptance criteria called for testing that "Cannot" (capital C) is used. Since the implementation uses lowercase (correctly per Go convention), the test asserts lowercase, which is consistent.
- No separate test isolates just the second line or just the capitalization -- but given the exact string match, this is implicitly covered.

### Spec Compliance

**Spec reference** (lines 405-408 of `tick-core.md`):
```
Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-epic
       (would create unworkable task due to leaf-only ready rule)
```

**Implementation output** (via CLI layer):
```
Error: cannot add dependency - tick-child cannot be blocked by its parent tick-parent
(would create unworkable task due to leaf-only ready rule)
```

| Aspect | Spec | Implementation | Match |
|--------|------|----------------|-------|
| Two-line message | Yes | Yes | Yes |
| First line content | Correct | Correct | Yes |
| Second line rationale | Present | Present | Yes |
| Capitalization | "Cannot" | "cannot" | No (Go idiom wins) |
| Second line alignment | Indented | Flush left | Minor divergence |

The substantive content matches. The capitalization and alignment differences are defensible: lowercase errors follow Go convention, and alignment is a presentation concern that could be handled at the CLI formatting layer.

### golang-pro Compliance

| Requirement | Status |
|------------|--------|
| Errors explicitly handled | Yes |
| Error wrapping with `fmt.Errorf` | Yes (uses `%s` interpolation, not wrapping -- appropriate since this is a leaf error, not wrapping another) |
| Table-driven tests with subtests | Yes (existing test structure uses `t.Run`) |
| Exported functions documented | Yes (all exported functions have doc comments) |
| No panic for error handling | Yes |
| No ignored errors | Yes |

## Quality Assessment

### Strengths

1. **Surgical precision**: A 2-line code change plus 1-line test update. Exactly the right scope for an error message fix.
2. **Correct Go idiom over spec literalism**: Keeping lowercase error strings despite the task plan requesting uppercase shows good engineering judgment.
3. **Proper layer separation**: The error message contains only the semantic content; the "Error: " prefix is correctly left to the CLI dispatcher.
4. **Test updated in lock-step**: The assertion was updated to match the new message exactly.

### Weaknesses

1. **Minor spec divergence on alignment**: The second line lacks the leading whitespace shown in the spec. If the spec's indentation is intentional (to visually align under the first line after "Error: "), the implementation does not reproduce this. This is cosmetic but could matter for agent parsing.
2. **No dedicated test for the second line**: The task plan's acceptance criteria listed three separate test conditions (both lines present, capital C, rationale text). These are all covered implicitly by the single exact-match assertion, but dedicated sub-assertions or a separate subtest would make intent clearer.
3. **Task plan acceptance criteria partially unmet**: The plan said to use "Cannot" (capital C). While the decision to keep lowercase is correct per Go idiom, the task plan should have been updated to reflect this deviation. The plan was marked completed without noting the intentional divergence.

### Overall Rating

**8/10** -- A clean, minimal, correct change that adds the required explanatory line. The implementation correctly prioritizes Go conventions over a literal spec reading on capitalization. The only material concern is the minor alignment divergence from the spec's visual layout, and the lack of documentation about the intentional capitalization deviation from the task plan.
