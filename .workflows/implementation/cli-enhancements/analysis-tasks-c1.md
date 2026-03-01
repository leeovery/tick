---
topic: cli-enhancements
cycle: 1
total_proposed: 5
---
# Analysis Tasks: cli-enhancements (Cycle 1)

## Task 1: Add type column to queryShowData SQL query
status: pending
severity: high
sources: standards, architecture

**Problem**: The `showData` struct in `internal/cli/show.go` has no `type` field, the SQL SELECT at line 73 omits `type`, and `showDataToTaskDetail` never sets `Task.Type`. This causes all detail output (show, create, update, note) to render the type as empty/dash regardless of the task's actual type value. All three formatters (Pretty, Toon, JSON) are affected. Existing tests pass only because they create tasks without a type, so the empty default matches expectations.
**Solution**: Add a `taskType string` field to `showData`, add `type` to the SQL SELECT and Scan in `queryShowData`, and populate `Task.Type` in `showDataToTaskDetail`.
**Outcome**: `tick show`, post-create, post-update, and post-note output all correctly display the task's type when set. All three formatters render the type value.
**Do**:
1. Add `taskType string` field to the `showData` struct (after `priority` to match column order).
2. In `queryShowData`, add `type` to the SQL SELECT clause: `SELECT id, title, status, priority, type, description, parent, created, updated, closed FROM tasks WHERE id = ?`.
3. Add a `*string` pointer variable (e.g., `typePtr`) alongside `descPtr`/`parentPtr`/`closedPtr`, and include it in the `Scan` call in the correct positional order.
4. After the Scan, set `data.taskType = *typePtr` when non-nil (same pattern as description/parent/closed).
5. In `showDataToTaskDetail`, set `Type: d.taskType` on the `task.Task` literal.
6. Add a test case that creates a task with `--type bug`, runs `tick show`, and verifies the output contains `Type: bug` (or the Pretty/Toon/JSON equivalent).
**Acceptance Criteria**:
- `showData` struct includes a type field
- SQL SELECT in `queryShowData` includes `type` column
- `showDataToTaskDetail` populates `Task.Type` from `showData`
- `tick show` output displays the correct type for tasks that have one
- Post-mutation output (create, update, note) displays the correct type
**Tests**:
- Create a task with `--type bug`, run `tick show <id> --pretty`, verify output contains "Type: bug"
- Create a task without `--type`, run `tick show <id> --pretty`, verify output contains "Type: -" (or equivalent empty representation)
- Create a task with `--type feature`, run `tick update <id> --title newtitle --pretty`, verify post-mutation output contains "Type: feature"

## Task 2: Extract generic deduplication and validation helpers in internal/task
status: pending
severity: medium
sources: duplication

**Problem**: `DeduplicateTags` and `DeduplicateRefs` in `internal/task/` implement identical algorithms (iterate, normalize, skip empties, deduplicate by seen-map, preserve order) differing only in the normalizer function. `ValidateTags` and `ValidateRefs` follow the same pattern (early-return on empty, call deduplicate, validate each element, check max count). This is ~60 lines of near-duplicate logic across `tags.go` and `refs.go` that risks drift as features evolve.
**Solution**: Extract a `deduplicateStrings(items []string, normalize func(string) string) []string` helper. Make `DeduplicateTags` and `DeduplicateRefs` thin wrappers. Optionally extract a `validateCollection` helper for the shared validate pattern.
**Outcome**: Tags and refs deduplication/validation share a single implementation. Adding a new slice-type field in the future requires only a wrapper, not a copy.
**Do**:
1. In `internal/task/`, create (or add to an existing shared file) a `deduplicateStrings(items []string, normalize func(string) string) []string` function implementing the current dedup algorithm.
2. Rewrite `DeduplicateTags` as: `return deduplicateStrings(tags, NormalizeTag)`.
3. Rewrite `DeduplicateRefs` as: `return deduplicateStrings(refs, strings.TrimSpace)`.
4. Verify all existing tests in `internal/task/` still pass.
5. Optionally: extract a `validateCollection` helper that takes the deduplicate function, per-item validator, and max count, then rewrite `ValidateTags`/`ValidateRefs` as wrappers.
**Acceptance Criteria**:
- `DeduplicateTags` and `DeduplicateRefs` delegate to a shared helper
- No behavioral change -- all existing tests pass without modification
- The shared helper is unexported (internal implementation detail)
**Tests**:
- All existing tag and ref deduplication tests pass unchanged
- All existing tag and ref validation tests pass unchanged

## Task 3: Extract shared buildStringListSection in toon_formatter.go
status: pending
severity: medium
sources: duplication

**Problem**: `buildTagsSection` and `buildRefsSection` in `internal/cli/toon_formatter.go` are structurally identical 8-line functions that differ only in the section name string ("tags" vs "refs"). This is minor duplication but represents a pattern that would multiply with each new list-type field.
**Solution**: Extract a `buildStringListSection(name string, items []string) string` function. Replace both `buildTagsSection` and `buildRefsSection` with calls to it.
**Outcome**: One function handles all string-list TOON sections. Adding future list fields requires no new formatter function.
**Do**:
1. In `toon_formatter.go`, add an unexported function: `func buildStringListSection(name string, items []string) string` implementing the current shared pattern (write "name[count]:", iterate and write "\n  " + value).
2. Replace `buildTagsSection` body with `return buildStringListSection("tags", tags)`.
3. Replace `buildRefsSection` body with `return buildStringListSection("refs", refs)`.
4. Run `go test ./internal/cli/` to verify no regressions.
**Acceptance Criteria**:
- `buildTagsSection` and `buildRefsSection` delegate to a single shared function
- TOON formatter output is identical before and after (no behavioral change)
**Tests**:
- All existing toon formatter tests pass unchanged
- Verify `tick show` with --toon on a task with tags and refs produces identical output

## Task 4: Extract shared validation helpers for type/tags/refs flags in CLI layer
status: pending
severity: medium
sources: duplication

**Problem**: `RunCreate` (create.go:130-160) and `RunUpdate` (update.go:159-202) contain nearly identical validation blocks for `--type`, `--tags`, and `--refs` flags. Each independently implements normalize-then-validate-then-check-empty sequences. This is 6 blocks of structurally identical validation (~5-8 lines each) spread across two files.
**Solution**: Extract `validateTypeFlag(value string) (string, error)`, `validateTagsFlag(tags []string) ([]string, error)`, and `validateRefsFlag(refs []string) ([]string, error)` into `helpers.go` (or a new `validate_helpers.go`). Each encapsulates the normalize + empty-check + validate sequence.
**Outcome**: Validation logic for these three fields lives in one place. create.go and update.go each call one-line helpers instead of repeating the same multi-step sequences.
**Do**:
1. In `internal/cli/helpers.go` (or a new `validate_helpers.go`), add three functions:
   - `validateTypeFlag(value string) (string, error)` -- normalize (TrimSpace, ToLower), check empty, call `task.ValidateType`.
   - `validateTagsFlag(tags []string) ([]string, error)` -- call `task.DeduplicateTags`, check all empty, call `task.ValidateTags`.
   - `validateRefsFlag(refs []string) ([]string, error)` -- call `task.DeduplicateRefs`, check all empty, call `task.ValidateRefs`.
2. In `create.go`, replace the inline type/tags/refs validation blocks with calls to these helpers.
3. In `update.go`, replace the inline type/tags/refs validation blocks with calls to these helpers.
4. Run `go test ./internal/cli/` to verify no regressions.
**Acceptance Criteria**:
- Validation logic for type, tags, and refs flags is defined once in helpers
- create.go and update.go use the shared helpers
- No behavioral change -- all existing create and update tests pass
**Tests**:
- All existing create tests pass unchanged
- All existing update tests pass unchanged
- Verify that invalid type/tags/refs still produce the same error messages

## Task 5: Consolidate ResolveID into a single Query call
status: pending
severity: medium
sources: architecture

**Problem**: `ResolveID` in `internal/storage/store.go:272-329` calls `s.Query()` up to twice -- once for exact full-ID match and once for prefix search fallback. Each `Query()` call acquires a shared lock, reads the JSONL file, checks cache freshness, then releases the lock. This doubles I/O and lock overhead for every full-ID resolution, and `ResolveID` is called on nearly every command.
**Solution**: Refactor `ResolveID` to perform both the exact match attempt and the prefix search within a single `s.Query()` call.
**Outcome**: Every ID resolution uses one lock acquisition and one cache-freshness check instead of two, reducing per-command latency.
**Do**:
1. In `store.go`, refactor `ResolveID` so that the exact-match check and the prefix-search fallback both happen inside a single `s.Query(func(db *sql.DB) error { ... })` block.
2. Inside the single callback: attempt exact match first; if not found and prefix length < 6, perform the LIKE query; handle ambiguity and not-found errors as before.
3. Run `go test ./internal/storage/ ./internal/cli/` to verify no regressions.
**Acceptance Criteria**:
- `ResolveID` makes exactly one `s.Query()` call regardless of whether it does exact or prefix matching
- All existing ResolveID tests pass unchanged
- Ambiguity and not-found error behavior is preserved
**Tests**:
- All existing ResolveID/partial-ID tests pass unchanged
- Exact full-ID resolution still works correctly
- Prefix resolution with ambiguity still returns error listing matching IDs
