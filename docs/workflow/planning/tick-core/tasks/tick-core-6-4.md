---
id: tick-core-6-4
phase: 6
status: completed
created: 2026-02-10
---

# Consolidate formatter duplication and fix Unicode arrow

**Problem**: Two issues affecting the same code. (1) `ToonFormatter.FormatTransition` and `PrettyFormatter.FormatTransition` have identical implementations. Same for `FormatDepChange` -- four methods total are exact copies across `internal/cli/toon_formatter.go` and `internal/cli/pretty_formatter.go`. (2) All transition formatters use ASCII `->` but the spec (line 639) specifies Unicode right arrow (U+2192). The codebase is internally inconsistent: dependency.go cycle errors correctly use `\u2192`.

**Solution**: Extract a shared base implementation (e.g., `baseFormatter` embedded struct) providing `FormatTransition` and `FormatDepChange`. Both ToonFormatter and PrettyFormatter embed it. Fix the arrow character to use Unicode `\u2192` in FormatTransition, matching the spec and dependency.go.

**Outcome**: Four duplicate methods consolidated to two. Transition output uses the spec-correct Unicode arrow. Internal consistency with dependency.go cycle error messages.

**Do**:
1. Create a `baseFormatter` struct in an appropriate file (e.g., `internal/cli/format.go`)
2. Move `FormatTransition` and `FormatDepChange` to methods on `baseFormatter`, changing `->` to the Unicode arrow `\u2192` in FormatTransition
3. Embed `baseFormatter` in both `ToonFormatter` and `PrettyFormatter`
4. Remove the now-redundant methods from `toon_formatter.go` and `pretty_formatter.go`
5. Check `JsonFormatter` -- if it also has identical implementations, embed there too
6. Update any tests that assert on the ASCII `->` arrow to use the Unicode arrow

**Acceptance Criteria**:
- `FormatTransition` and `FormatDepChange` exist in one place only
- Transition output uses Unicode right arrow matching spec line 639
- ToonFormatter, PrettyFormatter (and JsonFormatter if applicable) produce correct output
- All existing formatter tests pass

**Tests**:
- Test that FormatTransition output contains the Unicode arrow character
- Test that FormatTransition output matches spec format: `tick-id: old_status -> new_status`
- Test that FormatDepChange output is correct for add and remove cases
- Test that all three formatters produce consistent transition output
