TASK: free-text-round-trip-7-5 (tick-5087a7) — Corrections: delete `TestConformanceScopeBoundary`, a guard asserting over the source text of four sibling test files

ACCEPTANCE CRITERIA:
- [ ] `grep -rn 'TestConformanceScopeBoundary' internal/cli` returns nothing.
- [ ] `internal/cli` compiles with no unused import and `go vet ./...` is clean.
- [ ] The four files the guard named are byte-unchanged.
- [ ] `go test ./...` green, with no other test deleted, renamed or weakened.

STATUS: complete

SPEC CONTEXT: §11 "Conformance Verification" sets three parts — every structured command's output decoded by a real TOON reader (part 1), the awkward-task round-trip fixture (part 2), and "Rewritten assertions check decoded values, not output text" (part 3) — plus "No byte-level pinning is kept in the machine formats", with pretty exempted because it has no parser. Nothing in §11 asks for a guard over test-file source text; the deleted test pinned identifiers in sibling sources, which is the opposite of part 3's "decoded values, not output text". The replacement guard §11 part 1 names is already in place: the toon and JSON drivers run the whole inventory, and `TestConformanceInventoryCoversEveryCommand` (internal/cli/conformance_test.go:1652) holds the inventory against the command registry, so an assertion lost where it mattered surfaces as a document nobody checks.

IMPLEMENTATION:
- Status: Implemented
- Location: commit a8305929, `internal/cli/conformance_test.go` — 33 deletions, no other file touched. The function and its single subtest are gone; the file's import block (internal/cli/conformance_test.go:3-18) now carries `bytes`, `encoding/json`, `errors`, `fmt`, `io`, `maps`, `slices`, `strings`, `testing`, `time`, `toon`, `task`, with `os`, `path/filepath` and `github.com/leeovery/tick/internal/testutil` dropped exactly as the task prescribed.
- Notes: Verified against each criterion.
  1. `grep -rn 'ScopeBoundary'` across the repo's Go sources returns nothing; the only surviving hit anywhere is the plan record in `.tick/tasks.jsonl:232`, outside `internal/cli`.
  2. No import is orphaned: every one of the twelve remaining imports has at least one use in the file (`os.`, `filepath.` and `testutil.` now have zero, and are absent from the block). `strings` correctly stays — 8 use sites remain.
  3. `git diff --stat a8305929^ a8305929` shows `internal/cli/conformance_test.go | 33 ------` and nothing else, so `toon_decode_test.go`, `toon_formatter_test.go`, `stats_test.go` and `dep_tree_test.go` were byte-unchanged by this task. (Later tasks in phases 8-10 edited three of them — outside this task's scope.)
  4. The diff removes no test other than the guard, and renames or weakens none.
  Removing the `testutil` import orphans nothing: `testutil.FindRepoRoot` keeps six other callers (`cmd/tick/build_test.go:15`, `internal/cli/readme_samples_test.go:51`, `scripts/naming_contract_test.go:105`, `scripts/install_test.go:18`, `scripts/release_test.go:48`, plus its own test at `internal/testutil/reporoot_test.go`).
  The deletion site (internal/cli/conformance_test.go:1298-1317) is clean — `assertConformanceKeys` and `assertConformancePriorityCounts` now sit adjacent with no residue, dangling comment or orphaned helper.

TESTS:
- Status: Adequate
- Coverage: This task deletes a guard that exercised no product behaviour, so no test is owed and none was added — correct. The coverage the guard claimed to protect all still exists and is unedited by this task: `TestToonTaskDetailConformance` (internal/cli/toon_decode_test.go:184), `TestToonDepTreeFocusedConformance` (internal/cli/toon_decode_test.go:402), `"it emits stats counts as top-level named fields"` (internal/cli/toon_formatter_test.go:335), `"it emits the dep tree summary as top-level named fields"` (internal/cli/toon_formatter_test.go:977), `"it decodes stats for a project with no tasks"` (internal/cli/stats_test.go:290) and `"it returns the emptied document when no task has dependencies"` (internal/cli/dep_tree_test.go:352). The real guard — `TestToonOutputConformance` (internal/cli/conformance_test.go:694) and `TestJSONOutputConformance` (:1692) driving the inventory, with `TestConformanceInventoryWellFormed` (:839) and `TestConformanceInventoryCoversEveryCommand` (:1652) holding the inventory itself — is untouched.
- Notes: The stated Outcome holds today: no `*_test.go` in the repo reads another `*_test.go` as source text (grep for `_test.go` inside test files returns no hit outside the deleted code).

CODE QUALITY:
- Project conventions: Followed. The deletion respects CLAUDE.md's stdlib-only, `t.Run()` subtest conventions and leaves the file's structure intact; nothing in the suite now asserts over source text, which matches §11's decoded-value rule.
- SOLID principles: Good — removing a test that coupled the conformance file to four unrelated files' source text drops a dependency the suite had no reason to carry.
- Complexity: Low — a pure deletion.
- Modern idioms: Yes.
- Readability: Good.
- Issues: None. The changed file carries no process-artifact references (no "phase N", no "§N", no task ids) after the deletion.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`internal/cli` compiles with no unused import and `go vet ./...` is clean." — the unused-import half is settled by reading (all twelve remaining imports have use sites; `os`, `path/filepath` and `testutil` are absent and unreferenced). The compile and vet halves need `go build ./...` and `go vet ./...` run.
- "`go test ./...` green, with no other test deleted, renamed or weakened." — the second half is settled by reading the commit diff (33 deletions, all inside the removed guard, no other test touched). The green half needs `go test ./...` run.
