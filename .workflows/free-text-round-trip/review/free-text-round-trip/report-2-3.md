TASK: free-text-round-trip-2-3 (tick-8daf62) — JSON Status Output Becomes The Changed List

ACCEPTANCE CRITERIA:
- `tick start|done|cancel|reopen --json` emits a single object whose only key is `changed`
- `changed` is `[]` and never `null` when nothing moved
- Each element carries `id`, `title`, `from`, `to` and `auto`, with `auto` a JSON boolean (`true`/`false`, unquoted)
- The single-transition branch and the cascade branch produce the same shape, differing only in row count
- Row order and row content match the toon table for the same command
- `grep -rn '"transition"\|"cascaded"\|jsonCascadeResult\|jsonCascadeTransition\|jsonCascadeEntry' internal/` returns nothing
- Toon and pretty output are unchanged by this task
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT:
§7.2 fixes the status-change row's vocabulary as `id`, `title`, `from`, `to`, `auto`, with `auto` false only for the change the caller asked for, the title carried "so no second lookup is needed to know what moved", each task appearing at most once, and a task that ends where it started carrying no row. §7.3 requires `done`, `start`, `cancel` and `reopen` to return *only* the `changed` table. §4.2 requires JSON to move with toon so a consumer parsing JSON gets the same structured answer as one parsing toon — the `changed` list in place of the `transition` object beside a `cascaded` list. The JSON formatter's existing `toJSONRelated`/`toJSONStrings` convention (non-nil empty slice so JSON renders `[]`, not `null`) is the one the new list follows.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/json_formatter.go:292-299` — `jsonStatusChange` carries `id`, `title`, `from`, `to` as strings and `Auto bool \`json:"auto"\``, so `auto` marshals as an unquoted JSON boolean.
  - `internal/cli/json_formatter.go:301-309` — `toJSONStatusChanges` allocates with `make([]jsonStatusChange, 0, len(changes))` and converts each `StatusChange` by struct conversion (the same shape as the neighbouring `jsonRemovedTask(r)` conversion at `internal/cli/json_formatter.go:282`), so an empty set marshals to `[]` rather than `null`.
  - `internal/cli/json_formatter.go:311-318` — `jsonChangedList` has the single field `Changed []jsonStatusChange \`json:"changed"\``; `FormatCascadeTransition` marshals it from `result.Changed()` with no early return, so an empty result renders `{"changed": []}`.
  - `internal/cli/format.go:208-228` and `internal/cli/transition.go:139-147` — the rows come from `CascadeResult.Changed()`, which feeds the primary change plus each cascade entry into `statusChangeSet` and drops any row whose `From == c.To`. An empty `CascadeResult{}` therefore yields a zero-length, non-nil slice (`From` and `To` both `""`), which is what makes the `{"changed": []}` branch fall out of the single code path rather than needing a special case.
  - `internal/cli/toon_formatter.go:158-162` and `internal/cli/json_formatter.go:317-318` — both formatters render `result.Changed()`, so row order and row content are the same rows by construction, not by parallel code kept in step.
  - `internal/cli/transition.go:16-59` — all four status commands share `RunTransition`, whose only non-quiet output is `outputStatusChanges` (`internal/cli/helpers.go:109-112`), so the whole stream for `start|done|cancel|reopen --json` is the one `{"changed": …}` object.
- Notes: the removal grep from the acceptance criteria (`'"transition"\|"cascaded"\|jsonCascadeResult\|jsonCascadeTransition\|jsonCascadeEntry'` over `internal/`) returns nothing; `jsonTransition` and `FormatTransition` are likewise absent from `internal/`, and `README.md`/`CLAUDE.md` carry no reference to the old keys. The production diff for this task (`be7c7371`) touches only `internal/cli/json_formatter.go`, so toon and pretty output are untouched.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/cascade_formatter_test.go:182-232` — the three formatter-level subtests: one-element list for a single transition (`auto` false), two-element list for a cascade (requested row first, second `auto` true), and an empty `CascadeResult` decoding to a non-nil zero-length `[]any`.
  - `internal/cli/cascade_formatter_test.go:144-154` — `decodeChangedDoc` asserts the decoded document has exactly one key, which is a stronger form of "no `transition` or `cascaded` key" than naming the two keys, and it gates every JSON assertion in the task (unit, integration and end-to-end all route through `changedRows`).
  - `internal/cli/cascade_formatter_test.go:166-180` — `assertChangedRow` checks each expected field by decoded value and separately asserts `got["auto"]` is a Go `bool`, so a quoted `"true"` would fail.
  - `internal/cli/transition_test.go:651-764` — end-to-end through `App.Run` with `--json`: one-element list, cascade list, title on every row read against the stored titles, and a row-for-row comparison against the same command's `--toon` output for `start`, `done` and `cancel`.
  - `internal/cli/format_test.go:415-418` — the empty-`CascadeResult` expectation moved off `""` for both toon and JSON, as the task's edge case required.
  - `internal/cli/format_integration_test.go:121-131` — the pre-existing `--json` transition case now reads the `changed` row by decoded value instead of the old `transition` object.
  - `internal/cli/conformance_test.go:399-462, 700-706, 773-789, 1692-1696` — all four status commands (cascading and non-cascading) run under the JSON conformance driver, which rejects a null `changed` and a non-boolean `auto` anywhere in the document. This is what covers `reopen --json`, which the row-for-row subtest does not enumerate.
- Notes: not over-tested — the formatter-level and end-to-end layers assert different things (shape and field values vs. real command output and toon parity), and no assertion pins a whole document string. `auto == true` on a system-initiated *primary* is covered where the merge is defined (`internal/cli/status_change_test.go:72`) rather than duplicated here, which is right: the JSON layer copies the bool.

CODE QUALITY:
- Project conventions: Followed. The non-nil-slice helper mirrors `toJSONStrings`/`toJSONRelated` (`internal/cli/json_formatter.go:150-155`), the struct conversion mirrors `jsonRemovedTask`, and every new type carries a doc comment in the file's existing voice.
- SOLID principles: Good. The formatter renders rows and adds no merging logic of its own; `CascadeResult.Changed()` remains the single producer shared with toon.
- Complexity: Low. One loop, one wrapper struct, no branches.
- Modern idioms: Yes. `make` with capacity, struct conversion instead of a field-by-field copy.
- Readability: Good. The doc comments state what the code does and hold true against it — `toJSONStatusChanges` says "Always returns a non-nil empty slice" and does, `jsonChangedList` says it holds every status change a command made and does. No comment references a task id, phase or spec section.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settled only by running the three commands from the repo root. Reading confirms the package is self-consistent (the `encoding/json` import in `internal/cli/format_integration_test.go` is still used at six sites after the rewrite removed one, and `maps`/`slices` are imported where `internal/cli/cascade_formatter_test.go:151` uses them), but neither compilation nor formatting can be judged by reading.
