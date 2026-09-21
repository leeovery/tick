TASK: free-text-round-trip-5-7 (tick-3f0af1) — Every Command Gets Its Arguments Split At The Marker

ACCEPTANCE CRITERIA:
1. `tick migrate --from beads -- --dry-run` performs a real import: the tasks land in `tasks.jsonl` and the dry-run presenter is not used.
2. Post-marker `--from`, `--pending-only` and `--dry-run` are text on migrate: `tick migrate -- --from beads` fails with `--from flag is required`, and a post-marker second `--from` does not override the pre-marker one.
3. `grep -rn 'splitLiteralArgs(' internal/ --include='*.go' | grep -v _test` returns exactly two lines — the definition in `flags.go` and the single call in `App.Run`.
4. `grep -rn 'SplitLiterals' internal/ --include='*.go'` returns nothing: the method, the `Literals` field and their test are gone.
5. Every handler and `Run*` that takes command arguments takes both halves; none re-derives the boundary from a count.
6. Marker behaviour on every other command is unchanged: `TestEndOfFlagsMarker`, `TestPostMarkerArgumentsAreNeverFlags`, `TestParseArgsLiterals` and `TestSplitLiteralArgs` pass as written, and `go test ./...` is green.

STATUS: complete

SPEC CONTEXT: §10.2 sets the rule this task enforces — "Nothing after the marker is read as a flag — not the command's own flags and not the global format flags — so flags come before it and everything after it is text" (specification.md:468). §10.1 frames the defect: `ValidateFlags` inspects arguments that are free text by definition, the one hole in §2.2's round-trip guarantee. §3.3 puts `migrate`'s *output* out of scope but keeps its *input* write path in (specification.md:82-90), so the marker applying to `migrate` invocations follows the spec rather than widening it. The README and `tick help` publish the rule as universal (README.md:641, help.go:272 and :281), which is what made `migrate` ignoring it a live trap.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - Single split: `internal/cli/app.go:43` — `flagArgs, literals := splitLiteralArgs(subArgs, flags.literals)`, computed immediately after `parseArgs`.
  - Validation sites reuse that one pair: `app.go:73` (doctor), `app.go:80` (migrate), `app.go:116-117` (`qualifyCommand(subcmd, flagArgs)` feeding `ValidateFlags`). The three validation-only splits are gone.
  - Migrate honours the marker: `app.go:84` passes `(flagArgs, literals)` to `handleMigrate` (`migrate.go:72`), which hands them to `parseMigrateArgs(flagArgs, _ []string)` (`migrate.go:47`) — the literals are dropped as `parseListFlags` does (`list.go:37`), migrate carrying no positional.
  - 12 handlers take both halves: `app.go:165` init (both blank), `:174` create, `:183` list, `:196` show, `:205` update, `:214` ready, `:227` blocked, `:261` remove, `:310` transition, `dep.go:13`, `note.go:15`, `migrate.go:72`.
  - 9 `Run*` take both halves: `create.go:112`, `show.go:37`, `update.go:168`, `transition.go:16`, `dep.go:57`, `dep.go:135`, `dep_tree.go:13`, `note.go:41`, `note.go:94`.
  - Sub-subcommand routed off the flag half: `dep.go:23-24` and `note.go:25-26` take `flagArgs[0]` and pass `flagArgs[1:]` with `literals` unchanged. Fully-positional commands concatenate in order via `slices.Concat(flagArgs, literals)` — `transition.go:17`, `dep.go:58`, `dep.go:136`, `dep_tree.go:14`, `note.go:42`, `note.go:95`.
  - `FormatConfig.SplitLiterals` and `FormatConfig.Literals` removed (`format.go:66-72` now carries Format/Quiet/Verbose/Logger only); `splitLiteralArgs` kept at `flags.go:181` with its one caller.
- Criteria settled by reading:
  - (1) MET. With `--dry-run` in the literal half, `parseMigrateArgs` never sees it, so `mf.dryRun` is false and `RunMigrate` (`migrate.go:107-118`) builds a `StoreTaskCreator` rather than `DryRunTaskCreator`; `migrate.Present` then omits the `[dry-run]` header (`internal/migrate/presenter.go:13`). Pinned by the subtest at `end_of_flags_test.go:105`.
  - (2) MET. `tick migrate -- --from beads` yields an empty `flagArgs`, so `parseMigrateArgs` returns `--from flag is required` (`migrate.go:65`). A post-marker second `--from` is never scanned, so it cannot override the pre-marker one. Pinned at `end_of_flags_test.go:126` and `:144`.
  - (3) MET. `grep -rn 'splitLiteralArgs(' internal/ --include='*.go' | grep -v _test` → exactly two lines: `flags.go:181` (definition) and `app.go:43` (call).
  - (4) MET. `grep -rn 'SplitLiterals' internal/ --include='*.go'` → no output. `TestFormatConfigSplitLiterals` is gone: `grep -rn 'func TestFormatConfig' internal/cli` returns only `format_test.go:129`, which covers verbosity/quiet, not splitting.
  - (5) MET. Every handler and `Run*` in the plan's enumeration takes `flagArgs, literals`; no site re-derives a boundary from a count, since no `Literals` field survives to read. `handleStats` (`app.go:240`), `handleRebuild` (`app.go:249`) and `handleDoctor` (`doctor.go:47`) take no argument slice.
- Notes (observations, not findings):
  - `app.go:65` (`a.handleHelp(subArgs)`) is the one remaining consumer of the unsplit slice. The plan's enumeration of 12 handlers deliberately excludes `handleHelp`: the pre-change line numbers it lists (166/175/184/198/207/216/230/265/315) map to init/create/list/show/update/ready/blocked/remove/transition in `18d1deb2^:internal/cli/app.go`, with `handleHelp` at 324 omitted. The only observable divergence is `tick help -- --all`, which prints the full reference rather than treating `--all` as a command name; `tick help -- create` still prints create help, the command name being a positional either way. Nothing is lost, and `help` is absent from `commandFlags`, so it never reaches `ValidateFlags`.
  - Deliberate, plan-directed behaviour change: `tick dep -- add a b` and `tick note -- add <id> text` previously routed (the sub-subcommand came off the unsplit slice, introduced one commit earlier in d8701f10) and now return "sub-command required", because routing reads `flagArgs[0]`. The task's Do list instructs this and it follows the §10.2 rule — a post-marker word is text, and text is not a command word. Nothing pins it either way, and no data path is lost.
  - `splitLiteralArgs`'s `n = min(n, len(args))` clamp is unreachable from its single caller now that no caller slices arguments off before splitting. Retained per the task's explicit instruction to keep the function and its test; the subtest at `end_of_flags_test.go:422` still pins it.

TESTS:
- Status: Adequate
- Coverage: All five tests the task names exist and assert what it asked for —
  - `end_of_flags_test.go:105` "it imports tasks when the marker precedes --dry-run on migrate": asserts the fixture task is in `readPersistedTasks` *and* that stdout carries no `[dry-run]`. This replaces the old absence-of-error-string guard, which stayed true while the marker was ignored; the new one fails if `--dry-run` ever leaks back into the flag half.
  - `:126` "it treats a post-marker --from as text on migrate": `--from beads -- --from bogus` still imports through the beads provider. Would fail with an unknown-provider error if the literal were scanned.
  - `:144` "it reports a missing --from when the only --from follows the marker": exit 1 carrying `--from flag is required`.
  - `:157` "it treats a post-marker --pending-only as text on migrate": two-issue fixture (one pending, one closed), asserts 2 imported. Fails with 1 if `--pending-only` leaked.
  - `:172` "it keeps the marker working on a command with a sub-subcommand": `note add <id> -- "- text"` stores the dash-leading text and `dep add -- <id> <blocker>` links them in order.
- Notes: One deliberate overlap, recorded rather than raised. The two halves of the subtest at `:172` each restate an existing subtest verbatim — the note half duplicates `:59` "it accepts dash-leading note text after the marker" (same command, same arguments, same assertion) and the dep half duplicates `:625` "it passes post-marker arguments to dep add in order". The task's Tests section asked for it by name, and it is the only subtest exercising both routing paths in one place after the `flagArgs[0]` change, so the redundancy buys a named regression guard. No excessive setup or mocking anywhere, and no test reaches into implementation detail — every assertion is made against persisted tasks or process output. Tests calling the changed parsers directly (`show_fields_test.go:16`, `:120`, `:190`, `:204`; `remove_test.go:1717-1862`) pass `nil` for the literal half and remain correct against the new signatures.

CODE QUALITY:
- Project conventions: Followed. Test naming uses the project's "it does X" subtest form; stdlib `testing` only; `t.Helper()` on `runTick`/`runMigrate`/`setupBeadsFixture`; `t.TempDir()`-backed fixtures. CLAUDE.md:48 was updated in the same commit to the new handler signature, so the repo's own reference no longer teaches the shape that silently drops post-marker text.
- SOLID principles: Good. The split has one owner (`App.Run`) instead of being an opt-in duty spread over two layers; handlers receive data rather than a mechanism, and `FormatConfig` no longer carries a boundary count, so formatting config and argument parsing stop sharing a type.
- Complexity: Low. A net removal of ten call sites; the only added construct is `slices.Concat` at six fully-positional entry points.
- Modern idioms: Yes — `slices.Concat`, `min`, and three-index slicing to clip `flagArgs` capacity (`flags.go:187`) so an `append` cannot overwrite the first literal.
- Readability: Good. The doc comments added to the six concatenating functions state the contract ("Every argument is positional, so the post-marker literals follow the flag half in order") rather than restating the line below them, and `parseMigrateArgs`'s comment explains why its second parameter is blank. `flags.go:177-180` had its now-false clause about counts taken before slicing removed. Every comment in the changed code holds against it.
- Issues: None rising to a finding. Two tensions recorded, both pre-existing or plan-directed: `golang-code-style` SKILL.md:174 sets a SHOULD of ≤4 parameters and the `Run*` functions now sit at 6 (`RunTransition` at 7) — they were already at 5-6 before this task, the plan directed the added parameter, and CLAUDE.md now documents the 6-parameter shape as the project's contract. `parseDepArgs` (`dep.go:40`) still copies with `append([]string{}, args...)` a slice its callers built fresh via `slices.Concat`; harmless and pre-existing.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "Marker behaviour on every other command is unchanged: `TestEndOfFlagsMarker`, `TestPostMarkerArgumentsAreNeverFlags`, `TestParseArgsLiterals` and `TestSplitLiteralArgs` pass as written, and `go test ./...` is green." — Reading settles the first half in substance: all four tests are present (`end_of_flags_test.go:35`, `:433`, `:339`, `:401`), the parser and split behaviour they assert is intact, and every caller of a changed signature matches the new shape by inspection. The green-suite half needs `go test ./...` executed at the repo root — in particular `internal/cli`, whose fixtures write real JSONL. Note that `TestEndOfFlagsMarker` is not literally "as written" pre-change: the task itself directed replacing the migrate guard at `end_of_flags_test.go:99-106` with the import assertion now at `:105`.
