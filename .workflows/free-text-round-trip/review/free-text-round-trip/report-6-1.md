TASK: free-text-round-trip-6-1 — Task-List Documents Decode Through One Conformance Table

ACCEPTANCE CRITERIA:
- `TestToonOutputConformance` decodes the `--toon` stdout of every non-exempt entry in `conformanceDocs` via the project's TOON library
- A decode failure reports the entry's `Name` and the undecodable document text
- The inventory carries a `list` document for a populated project, an empty project and a filter matching nothing, plus populated and empty documents for `ready` and for `blocked`
- The empty-project and filter-matches-nothing entries both decode to a document whose `tasks` key is an empty list
- A row whose `type` is empty decodes with `type` equal to `""` and every other column in its own position
- The `list --quiet` entry is present in the inventory with a stated reason and is not decoded
- `TestConformanceInventoryWellFormed` fails when two entries share a `Name` or when an entry carries both a `Setup` and a `NotADocument` reason
- No assertion on a populated task-list document compares rendered output against a pinned string
- The count-zero `tasks` header's column schema is still asserted as text, and the same subtest additionally asserts the decoded empty list
- `internal/cli/blocked_test.go` and `internal/cli/list_filter_test.go` still assert pretty's `No tasks found.` against their existing golden strings
- `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

STATUS: complete

SPEC CONTEXT: §11 part 1 requires every structured command's output to be decoded by a real TOON reader in the suite, counted in documents rather than commands — every branch a listed command can take, the emptied forms of §8 included. §3.1's table names `list`, `ready` and `blocked` as the task-list commands. §11 part 3 and "no byte-level pinning is kept in the machine formats" require rewritten assertions to check decoded values, while pretty keeps its golden strings because it has no parser. §12.1 records the empty-`type` case: the formatter quotes an empty type as `""`, which is the render the README got wrong.

IMPLEMENTATION:
- Status: Implemented (later tasks in the phase extended the same structures; the extensions are additive and consistent with this task's shape)
- Location:
  - `internal/cli/conformance_test.go:27` — `conformanceDoc` with `Name`, `Command`, `Setup`, `NotADocument` (plus `ToonNotADocument`, added by task 6-8 for the per-format prose exemption)
  - `internal/cli/conformance_test.go:205-266` — the task-list entries: `list` populated (207), `list` on an empty project (215), `list --status done` matching nothing (223), `list --quiet` exempt with a stated reason (231), `ready` populated (236) and empty (244), `blocked` populated (252) and empty (260)
  - `internal/cli/conformance_test.go:582-593` — `runTickConformance` builds the `App` with `IsTTY: true`, so the format flag the driver passes is what resolves the format rather than the non-TTY default
  - `internal/cli/conformance_test.go:609-621` — `decodeConformanceDoc` names the entry and quotes the document on a decode failure
  - `internal/cli/conformance_test.go:694-698` — `TestToonOutputConformance` drives the whole inventory, skipping exempt entries and decoding each document
  - `internal/cli/conformance_test.go:839-950` — `TestConformanceInventoryWellFormed`
  - `internal/cli/conformance_test.go:955-971` — `TestConformanceDecodeFailureReporting`
  - `internal/cli/conformance_test.go:1321-1392` — `assertConformanceTaskRows` and `TestToonTaskListConformance`
  - `internal/cli/toon_formatter_test.go:70-90` — populated list subtest now decodes and asserts row by row; `:92-113` — the two count-zero subtests keep the `tasks[0]{id,title,status,priority,type}:` text comparison and add the decoded-empty-list assertion
- Notes: The `list --quiet` reason ("--quiet prints bare task IDs, one per line, rather than a document") matches the branch at `internal/cli/list.go:221-226`. The empty branch the task exists to cover is now `emptyToonSection[toonTaskRow]("tasks")` (`internal/cli/toon_formatter.go:53-55`) — a later task replaced the hand-written literal with a generic derived from the row struct, and the retained text assertion still pins the column list, so the criterion is met in substance. `runTickConformance`'s signature grew a `format` parameter when task 6-5 added the JSON driver; the `--toon` path is unchanged.

TESTS:
- Status: Adequate
- Coverage: All thirteen micro-acceptance tests named in the plan are present. Parseability is covered generically by `TestToonOutputConformance` over the inventory; values are covered per document by `TestToonTaskListConformance` (`conformance_test.go:1338-1392`), which asserts every column of every row in order via `assertConformanceTaskRows` — so a column shift or a dropped field fails, not merely an unparseable document. `ready` expects only the blocker (`conformanceBlockedPair()[:1]`) and `blocked` only the dependent (`[1:]`), so each handler's own filter is exercised rather than borrowed from `list`. Row order matches `list`'s `ORDER BY (t.status = 'in_progress') DESC, t.priority ASC, t.created ASC` (`internal/cli/list.go:317`) for the seeded fixture, so the positional assertions are deterministic. The well-formedness guard is exercised against constructed inventories (duplicate name, setup+reason, neither, command mismatch) as well as against the real `conformanceDocs`, and `conformance_test.go:1271-1293` proves an exempt entry's `Setup` is never run by the toon driver.
- Notes: `"it decodes an empty type as an empty string"` (`conformance_test.go:1345-1355`) re-asserts a row `assertConformanceTaskRows` already covers, but the plan names it and it pins §12.1's case where a reader will look for it. Pretty's `No tasks found.` goldens are untouched (`internal/cli/blocked_test.go:192,206,290`; `internal/cli/list_filter_test.go:299,400,477`), and no populated toon task-list assertion compares against a pinned string anywhere in `internal/cli` — the only remaining pinned row, `internal/cli/readme_samples_test.go:31`, asserts README content under §12.1, not rendered output.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests in "it does X" form, `t.Helper()` on every helper, `t.TempDir()` isolation via `setupTickProject`/`setupTickProjectWithTasks`.
- SOLID principles: Good — the inventory is data, the drivers are behaviour, and the decode/report step is separated into `decodeConformanceDoc`, which is why the failure-reporting test can assert on it without running a command.
- Complexity: Low. `inventoryProblems` collects problems rather than failing inline, which lets the same function check both the real inventory and the fixtures.
- Modern idioms: Yes — generics-free here but `slices.Equal`/`strings.Fields` in the command guard, `fmt.Errorf` with `%w` in the decoder.
- Readability: Good. The `conformanceDoc` doc comment states the `Setup`/`NotADocument` exclusivity the guard enforces, and every exemption reason names the actual branch.
- Comment accuracy: Comments hold against the code; no references to task ids, phases or spec sections anywhere in the two files.
- Issues: None worth reporting.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean" — settled only by running the four commands from the repo root; reading cannot establish that the suite passes or that the linters are silent.
