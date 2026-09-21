TASK: free-text-round-trip-4-9 (tick-6c3925) — Help Flags Print Their Description In Its Own Column

ACCEPTANCE CRITERIA:
- `tick help show` prints `  --field, --fields <name,...>  Select fields by name; a section may be narrowed with .N (e.g. notes.2)` — two spaces between label and description.
- `tick help --all` prints that same line for `show`.
- Every flag line in every command's help has at least two spaces between its label and its description, `list`'s 42-column `--status` and `create`'s 31-column `--type` included.
- The flag block a command renders under `tick help <cmd>` is byte-identical to the block it renders inside `tick help --all`.
- Label construction and the flag-line format string exist once in `internal/cli/help.go`; `grep -n '%-24s' internal/cli/help.go` returns nothing.
- A command with no flags still prints no `Flags:` section, and `go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass.

STATUS: complete

SPEC CONTEXT: §12.1 (specification.md:502-521) makes the command's own help text a deliverable of this work — `--field`/`--fields` has to be registered in the help registry for §9.6's unrecognised-name error to fire and for `TestCommandFlagsMatchHelp` to stay green, and the help is the surface an agent reads to discover the flag. A flag line whose argument spec and description are fused is undiscoverable, which is what this task removes.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/help.go:293-308 (`printFlagBlock`), called from `printAllHelp` at internal/cli/help.go:286 and `printCommandHelp` at internal/cli/help.go:319. Commit 3c286dd5.
- Notes:
  - `printFlagBlock` builds each label as `f.Name` plus `" " + f.Arg` when `Arg` is non-empty, takes `column = max(len(label)+2)` over the slice, and writes `"  %-*s%s\n"` — exactly the shape the task prescribed. Both renderers pass `cmd.Flags`, so the two blocks come from one code path and are byte-identical by construction.
  - `%-24s` is gone: `grep -n '%-24s' internal/cli/help.go` returns nothing, and the string appears nowhere else under `internal/`.
  - Column widths follow from the registry and match the task's figures: `show` label `--field, --fields <name,...>` is 28 → column 30 (exactly two spaces, confirmed by rendering the registry's description against the test's pinned line — byte-equal); `create`/`update` widest is `--type <bug|feature|task|chore>` at 31 → 33; `list`/`ready`/`blocked` widest is `--status <open|in_progress|done|cancelled>` at 42 → 44; `remove` `--force, -f` at 11 → 13; `migrate` `--from <provider>` at 17 → 19. No fixed floor, so narrow blocks narrow.
  - The top-level command column `"  %-14s%s\n"` (internal/cli/help.go:261) and the hand-written global-flags lines (internal/cli/help.go:265-272) are untouched, as instructed.
  - Zero-flag commands: `printCommandHelp` still gates the `Flags:` header on `len(cmd.Flags) > 0` (internal/cli/help.go:316); `printAllHelp` calls the helper unconditionally but an empty slice emits nothing.
  - Callers are complete — `printTopLevelHelp`/`printAllHelp`/`printCommandHelp` are invoked only from internal/cli/app.go:53, :321, :325, :333, so `tick create --help` takes the same path as `tick help create`.

TESTS:
- Status: Adequate
- Coverage: `TestHelpFlagColumn` (internal/cli/help_test.go:311-411) carries all five tests the task named:
  - "it separates every flag label from its description" (:312) loops every registry command with flags, asserts the flag line count matches the registry, that each line starts with `"  " + label`, that at least two spaces follow, and that the remainder equals the registered `Desc`. This is the criterion's own coverage, not a proxy.
  - "it prints the show field flag with its description in its own column" (:341) pins the exact `--field, --fields` line and asserts it in both `help show` and `help --all`.
  - "it renders the same flag block in --all as in per-command help" (:359) extracts the per-command block after `"Flags:\n"` and requires it verbatim inside `--all`, for every command carrying flags.
  - "it sizes the column to the widest label in the command's flag set" (:379) pins `create` at 33 and `remove` at 13 by reconstructing the expected line with `%-*s`.
  - "it prints no flags section for a command with no flags" (:402) uses `tick help init`.
- Notes:
  - The tests restate label construction in a local `flagLabels` helper (internal/cli/help_test.go:300-309) rather than calling `printFlagBlock`, so they would fail if the production label shape regressed — the right choice, not duplication worth collapsing.
  - Not over-tested: the four column tests each observe a different property (gap, exact line, cross-renderer identity, width sizing) and none re-pins a whole help document.
  - Existing `help_test.go` assertions are substring matches on flag names and usage lines, so the column shift leaves them green; no test anywhere else under `internal/` pins a spaced flag line (grep for `--flag` followed by three-plus spaces and a capital returns nothing outside `help_test.go`).

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests, `t.Helper()` on `helpFlagBlock`, "it does X" subtest naming.
- SOLID principles: Good — one helper owns the flag block; the two renderers keep their own responsibilities.
- Complexity: Low — two linear passes, no branching beyond the empty-`Arg` check.
- Modern idioms: Yes — builtin `max` (go 1.26 in go.mod) and `%-*s` dynamic width; `strings.CutPrefix` in the tests.
- Readability: Good — the doc comment on `printFlagBlock` ("descriptions aligned two spaces past the widest label in flags") holds true against the code.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "A command with no flags still prints no `Flags:` section, and `go test ./...`, `go vet ./...` and `golangci-lint run ./...` pass." — the no-flags half is settled by reading (internal/cli/help.go:316 gates the header; internal/cli/help_test.go:402 asserts it for `init`). The tool-run half needs those three commands executed at the current tree state.
