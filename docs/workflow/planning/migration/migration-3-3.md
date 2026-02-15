---
id: migration-3-3
phase: 3
status: completed
created: 2026-02-15
---

# Consolidate inconsistent empty-title fallback strings

**Problem**: Three locations use different fallback strings for empty titles: `engine.go:69` uses `"(untitled)"`, while `presenter.go:27-29` (`WriteResult`) and `presenter.go:64-66` (`WriteFailures`) both use `"(unknown)"`. These serve the same purpose but drifted because they were written independently.

**Solution**: Define a single exported constant (e.g., `FallbackTitle`) in `internal/migrate/migrate.go` and reference it from `engine.go`, `WriteResult`, and `WriteFailures`. Use `"(untitled)"` as the canonical value since it more accurately describes the situation (the title is missing, not the task's identity).

**Outcome**: One consistent fallback string used everywhere, defined in one place.

**Do**:
1. In `internal/migrate/migrate.go`, add: `const FallbackTitle = "(untitled)"`.
2. In `internal/migrate/engine.go:70`, replace `"(untitled)"` with `FallbackTitle`.
3. In `internal/migrate/presenter.go:28`, replace `"(unknown)"` with `FallbackTitle`.
4. In `internal/migrate/presenter.go:64`, replace `"(unknown)"` with `FallbackTitle`.
5. Update any tests that assert on the `"(unknown)"` string to use the new constant value.

**Acceptance Criteria**:
- A single `FallbackTitle` constant exists in `migrate.go`.
- All three usage sites reference the constant.
- No hardcoded fallback title strings remain in the migrate package.
- Tests pass with the consolidated value.

**Tests**:
- Existing tests for `WriteResult`, `WriteFailures`, and `Engine.Run` with empty-title tasks should pass after updating expected strings.
