TASK: free-text-round-trip-5-8 (tick-91fb2f) — A Flag Value Of Exactly Two Dashes Is Writable Again

ACCEPTANCE CRITERIA:
- `tick update <id> --title=--` and `tick create --description=-- "t"` succeed and store the two-dash value.
- `tick update <id> --description=--json` stores `--json` and does not switch the output format; the same value read out of `tick show --field description` and written back stores byte-identically.
- A dash-only or flag-spelling title and description survive the full round trip through `show --field` and back, asserted in the fixture rather than only at the parser.
- The separated form is unchanged on every flag: values, positionals, and the `… requires a value` error when a value-taking flag is the last flag argument.
- A positional carrying `=` is stored whole — `tick create -- "a=b"` stores the title `a=b`, `tick show a=b` resolves nothing rather than half an argument.
- `--clear-tags=x` and `--stauts=open` still fail with the existing unknown-flag error naming the whole argument; `--json=x` is still an unknown flag, since global flags stay exact-match in `applyGlobalFlag`.
- The attached form works on all five parsers: `tick list --status=open`, `tick show <id> --field=title`, `tick create --priority=0 "t"`, `tick update <id> --tags=a,b`, `tick migrate --from=beads`.
- `go test ./...` is green with the existing marker and round-trip suites unmodified apart from the fixture additions.

STATUS: complete

SPEC CONTEXT: §2.2 sets the fidelity bar at byte-identity — read a value out of `tick show`, write it back unchanged, and the stored value is byte-for-byte what it was. §10.2 (with its 2026-09-20 corrigendum, specification.md:549) records that recognising the end-of-flags marker at any position took away one previously working case — `--` as the *value* of a registered value-taking flag — and that the marker alone cannot restore it, because a post-marker argument is a positional and can never re-attach to the flag. The corrigendum names the attached form `--flag=value` as the repair, and notes it closes with it the pre-existing sibling case where a value spelling a global flag is stripped wherever it appears. §11.2 requires the round-trip guarantee to be pinned in the permanent fixture rather than only at the parser.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/flags.go:193-199` — `cutFlagValue` cuts a flag-shaped argument at its first `=`; an argument not beginning with `-` is returned whole with `attached` false, so a positional carrying `=` is never split.
  - `internal/cli/flags.go:205-243` — `flagScanner` (`next`, `value`); `value` returns the attached value when the argument carried `=`, otherwise consumes the following argument and reports `ok=false` when none follows.
  - `internal/cli/flags.go:153-161` — `ValidateFlags` keeps the numeric and `globalFlagSet` checks on the whole argument, then looks the registry up under the cut name; `!ok || (attached && !def.TakesValue)` produces the existing whole-argument error, and a `TakesValue` flag given attached no longer skips the following argument.
  - Parser conversion, all 24 value cases across exactly the five files the plan names: `create.go:42-102` (8), `update.go:49-114` (8), `list.go:39-102` (6), `show_fields.go:252-273` (1), `migrate.go:49-63` (1). Counted with `grep -rn 'requires a value' internal/cli/*.go | grep -v _test` → 24 lines in those five files only.
  - Positional paths keep the whole argument: `create.go:101`, `update.go:113`, `show_fields.go:267-272` all read `s.whole`, never the cut name.
  - Docs: `README.md:643` states the attached spelling and gives `tick update tick-a1b2 --title=-- --description=--json`; `CLAUDE.md` gains a "Flag value spellings" pattern entry.
- Notes:
  - Ordering holds: `ValidateFlags` runs before every parser on both dispatch paths (`app.go:72-82` for doctor/migrate, `app.go:113-117` for the rest), so an unknown attached flag is rejected before a parser could record it as a positional.
  - `applyGlobalFlag` (`app.go:424-425`) is still an exact-match switch, so `--description=--json` is never claimed as a global flag, while `--json=x` reaches `ValidateFlags` and fails as an unknown flag — both criteria as written.
  - Separated values carrying `=` are untouched: `flagScanner.value` returns the raw next argument (`flags.go:242`), never a cut one.
  - `ready`/`blocked` route through `parseListFlags` (`app.go:219`, `app.go:232`), so they inherit the attached form with no separate code path.
  - `remove` is correctly left on whole-argument matching (`remove.go:31-37`): it registers only `TakesValue: false` flags (`flags.go:84-87`), so `--force=x` stays an unknown flag.
  - `migrate_test.go:364` was flipped from "it rejects --from=value" to "it accepts --from=value" — the one pre-existing assertion of the old behaviour, and the only one; `grep -rn '\-\-[a-z-]*=' internal/cli/*_test.go` finds no other test asserting rejection of the attached spelling.

TESTS:
- Status: Adequate
- Coverage:
  - `internal/cli/attached_flag_value_test.go:19-208` carries all ten tests the plan lists plus three justified extras: `--stauts=x` following `--status=open` (guards the "do not skip the next argument" branch in `ValidateFlags`), `--json=x` (pins global-flag exact-match), and `--priority=0` / `--tags=a,b` (create and update in the attached form).
  - `internal/cli/round_trip_test.go:282-338` (`TestHostileValueRoundTrip`) pins the bar where it broke: `--` and `--json` on both title and description, each written in, asserted stored, read back with `show --field` (`bareField`, `round_trip_test.go:368-379`), written back through the attached form and re-asserted byte-identical.
  - The awkward fixture (`round_trip_test.go:82-280`) is otherwise unmodified apart from the shared `createCarryingValue` helper, as the plan required.
- Notes:
  - The tests discriminate. `assertNotJSON` (`attached_flag_value_test.go:212-218`) is meaningful because `runCreate`/`runUpdate` set `IsTTY: true` (`create_test.go:61-73`, `update_test.go:15-27`), so the output is pretty unless `--json` leaked to `applyGlobalFlag` — which is exactly the failure being excluded.
  - `--priority=0` is a discriminating value (the create default is 2, `create.go:40`), so the test fails if the attached value is dropped rather than parsed.
  - Not over-tested: the `--json` cases in `attached_flag_value_test.go:49-76` assert format non-switching, while the `TestHostileValueRoundTrip` `--json` cases assert read-back byte-identity — different guarantees, not a repeated happy path.
  - Test style matches the project: stdlib `testing`, `t.Run` "it does X" names, `t.Helper()` on helpers, `t.TempDir()`-backed `setupTickProject`.

CODE QUALITY:
- Project conventions: Followed. Handler signatures and the `flagArgs`/`literals` split are unchanged; the pattern is documented in CLAUDE.md alongside the existing flag-validation entry.
- SOLID principles: Good. `cutFlagValue` holds the one spelling rule; `flagScanner` holds iteration and value resolution; the five parsers keep their own value semantics. One rule, one site — `ValidateFlags` and every parser cut the argument through the same helper, so validation and parsing cannot drift on what counts as attached.
- Complexity: Low. `ValidateFlags` keeps its single loop with two added conditions; each parser's switch lost its index bookkeeping (`i+1 >= len(flagArgs)` / `i++`) to `s.value()`.
- Modern idioms: Yes — `strings.Cut` for the split, a pointer receiver iterator with `next`/`value`.
- Readability: Good. Field-level comments on `whole` and `name` (`flags.go:208-211`) state what each holds, and `value`'s doc (`flags.go:231-233`) states which argument it consumes.
- Comment accuracy: Comments hold. `ValidateFlags`' doc (`flags.go:120-128`) matches the `!ok || (attached && !def.TakesValue)` behaviour and the whole-argument error text; `cutFlagValue`'s doc matches its non-dash early return; `createCarryingValue`'s "the only spelling that survives a value of exactly the marker" (`round_trip_test.go:340-343`) is true — the separated form loses the value slot to the marker and a post-marker argument cannot re-attach.
- Security: No new surface — no shell, no interpolation; values are stored verbatim.
- Performance: No concern — one `strings.Cut` per flag-shaped argument.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...` is green with the existing marker and round-trip suites unmodified apart from the fixture additions." — settling it needs the suite run (and `go vet ./...` / `golangci-lint run ./...` for the lint half of the project's bar). Reading confirms the suites are unmodified apart from the fixture additions and the one flipped `migrate_test.go:364` assertion, and that no remaining test asserts the old rejection behaviour, but green is an execution result.
