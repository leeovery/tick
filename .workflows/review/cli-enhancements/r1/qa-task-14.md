TASK: cli-enhancements-3-4 -- Create and update with --tags and --clear-tags flags

ACCEPTANCE CRITERIA:
- `--tags <comma-separated>` on `create` and `update` sets/replaces all tags
- `--clear-tags` on `update` removes all tags
- `--tags` and `--clear-tags` are mutually exclusive
- Empty `--tags` value errors

STATUS: Complete

SPEC CONTEXT:
The specification defines `--tags <comma-separated>` on `create` and `update` to set/replace all tags, `--clear-tags` on `update` only to remove all tags, mutual exclusivity between the two, and empty `--tags` value producing an error. Tags are silently deduplicated, normalized (trimmed + lowercased), validated against kebab-case regex, with max 30 chars per tag and max 10 tags after dedup.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/create.go:79-85` -- `--tags` flag parsing in `parseCreateArgs`
  - `/Users/leeovery/Code/tick/internal/cli/create.go:138-143` -- tag validation in `RunCreate`
  - `/Users/leeovery/Code/tick/internal/cli/create.go:211` -- tags assigned to `newTask.Tags`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:25-26` -- `tags *[]string` and `clearTags bool` fields on `updateOpts`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:87-95` -- `--tags` and `--clear-tags` parsing in `parseUpdateArgs`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:172-181` -- mutual exclusivity check and validation in `RunUpdate`
  - `/Users/leeovery/Code/tick/internal/cli/update.go:277-281` -- tags applied to task in mutate callback
  - `/Users/leeovery/Code/tick/internal/cli/helpers.go:73-82` -- `validateTagsFlag` shared helper
  - `/Users/leeovery/Code/tick/internal/task/tags.go` -- domain validation (`ValidateTag`, `ValidateTags`, `DeduplicateTags`, `NormalizeTag`)
- Notes:
  - `--clear-tags` correctly absent from create command (only on update per spec)
  - `hasChanges()` at `/Users/leeovery/Code/tick/internal/cli/update.go:33` correctly includes `o.tags != nil || o.clearTags`
  - Error message for empty `--tags` on create: "cannot be empty; omit the flag to leave tags unset" -- appropriate guidance
  - Error message for empty `--tags` on update: "cannot be empty; use --clear-tags to remove all tags" -- appropriate guidance, context-aware
  - Mutual exclusivity checked before validation (correct order)
  - `--clear-tags` sets `tasks[i].Tags = nil` which is correct for `omitempty` JSON serialization

TESTS:
- Status: Adequate
- Coverage:
  - Create tests (`/Users/leeovery/Code/tick/internal/cli/create_test.go`):
    - Lines 882-901: creates task with `--tags ui,backend` (happy path, verifies persistence)
    - Lines 903-916: creates task without `--tags` (optional, no tags)
    - Lines 918-927: errors on empty `--tags` value
    - Lines 929-939: deduplicates tags on create (`ui,backend,ui` -> `[ui, backend]`)
    - Lines 941-957: normalizes tag input to lowercase (`UI,BACKEND` -> `[ui, backend]`)
    - Lines 959-968: rejects invalid kebab-case tag (`my tag`)
    - Lines 970-980: rejects more than 10 unique tags
  - Update tests (`/Users/leeovery/Code/tick/internal/cli/update_test.go`):
    - Lines 729-751: updates tags with `--tags api,frontend` (replaces existing `old-tag`)
    - Lines 753-769: clears tags with `--clear-tags` (removes `[backend, ui]`)
    - Lines 771-785: errors on `--tags` and `--clear-tags` together (mutually exclusive)
    - Lines 787-807: errors on empty `--tags` value on update (preserves existing tags)
    - Lines 809-825: `--clear-tags` on task with no tags succeeds (idempotent)
    - Lines 827-847: persists tags to JSONL and verifies output contains tags
    - Lines 849-876: `hasChanges` works for both `--tags` and `--clear-tags` independently
  - Domain-level tests (`/Users/leeovery/Code/tick/internal/task/tags_test.go`):
    - Full coverage of `ValidateTag`, `NormalizeTag`, `DeduplicateTags`, `ValidateTags`
    - Edge cases: double hyphens, leading/trailing hyphens, spaces, 30/31 char boundary, 11 tags deduped to 10, 11 unique tags
- Notes:
  - All edge cases from the plan task are covered: `--tags and --clear-tags together`, `empty --tags value`, `--tags with duplicates`, `--clear-tags on task with no tags`
  - Tests verify both the error condition and that the original data remains unchanged (e.g., empty --tags preserves existing tags)
  - Good separation: domain validation tested in `tags_test.go`, CLI integration tested in `create_test.go` and `update_test.go`

CODE QUALITY:
- Project conventions: Followed -- stdlib testing, `t.Run()` subtests, `t.Helper()`, `t.TempDir()` for isolation, error wrapping with `fmt.Errorf`
- SOLID principles: Good -- validation delegated to `task` package, shared `validateTagsFlag` helper in `helpers.go` avoids duplication, `createOpts`/`updateOpts` cleanly separate parsing from execution
- Complexity: Low -- straightforward flag parsing with switch-case, linear validation flow
- Modern idioms: Yes -- pointer types for optional field detection in `updateOpts`, comma-separated parsing, nil vs empty slice distinction
- Readability: Good -- clear naming, helpful error messages that guide users (e.g., "use --clear-tags to remove all tags"), comments on exported functions
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
