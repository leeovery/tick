# Consolidation Findings: Free Text Round Trip (Phase 5)

## Findings

### F1: `migrate` reads post-marker arguments as flags, against the rule this phase published

- **Class**: behaviour
- **Failure**: `tick migrate --from beads -- --dry-run` performs a dry run. `parseMigrateArgs` scans every argument it is given, so a post-marker `--dry-run` (or `--pending-only`, or a `--from` that overrides the first) still sets the flag, and nothing is imported. Meanwhile this phase published the opposite rule in three places: `tick help` (`help.go:272`), `tick help --all` (`help.go:281`) and the README, which now states "Flags come before `--`; everything after it is text, including an argument that spells a global flag exactly" (`README.md:641`) and lists `--` among the flags accepted on every command (`README.md:631`). A caller — an agent especially — that follows the documented rule and inserts the marker before a migrate invocation's arguments gets a silent no-op import and notices only when the tasks it expected are absent. The hole also reopens for every command added later: `App.Run` computes the split for validation (`app.go:79`) and then hands the *unsplit* args to the handler (`app.go:84`), and each handler opts into `fc.SplitLiterals` for itself, so forgetting the call in a new handler produces no compile error and no test failure. The phase's own guard, "it accepts the marker on migrate" (`end_of_flags_test.go:99-106`), asserts only that no unknown-flag error is printed — which stays true while the marker is ignored.
- **Evidence**:
  - `internal/cli/app.go:79-84` — the split half is computed for `ValidateFlags` and then discarded; `handleMigrate(subArgs)` receives the unsplit args
  - `internal/cli/migrate.go:46-65` — `parseMigrateArgs` scans all of `args` for `--from` / `--dry-run` / `--pending-only`
  - Seven per-site split calls, in two different layers: handlers `app.go:189` (list), `app.go:221` (ready), `app.go:235` (blocked), `app.go:271` (remove); `Run*` functions `create.go:115`, `update.go:171`, `show.go:38`
  - Three validation-only splits that drop the literals: `app.go:71` (doctor), `app.go:79` (migrate), `app.go:117` (everything else)
  - `internal/cli/help.go:272`, `internal/cli/help.go:281`, `README.md:631`, `README.md:641` — the published rule
  - `internal/cli/end_of_flags_test.go:99-106` — the migrate guard that passes either way
- **Proposed shape**: split once, above the handlers. `App.Run` computes `flagArgs, literals` immediately after `parseArgs` and hands both halves to every handler and `Run*` function, so a handler cannot forget the step and the validation call sites reuse the same pair. That removes the seven per-site calls and the three validation-only ones, puts the split in one layer instead of two, and makes `migrate` honour the marker as a consequence rather than as a second decision. Minimal alternative, if reaching every handler signature is judged too wide for this phase: give `parseMigrateArgs` the `(flagArgs, literals)` shape its siblings have and split in `handleMigrate` — migrate carries no positional, so the literals are dropped exactly as `parseListFlags` drops them (`list.go:37`). It is two lines, and it leaves the next handler free to forget. Either way, pin the behaviour with a guard that asserts the dry run did *not* happen (tasks imported) rather than asserting the absence of an error string.
- **Bank**: `free-text-round-trip-5-2` (reviewer) — "`migrate` is the one command still reading post-marker text as flags; the literal split is per-handler opt-in with nothing enforcing it." Confirmed against the final state: `internal/cli/migrate.go` is unchanged by the phase and `app.go:84` still passes `subArgs`.

### F2: a value-taking flag can no longer be given the value `--`

- **Class**: behaviour
- **Failure**: `tick update <id> --title --` and `tick create --description --` now fail with `--title requires a value` / `--description requires a value`. `parseArgs` consumes the first bare `--` wherever it appears, including the slot a value-taking flag's value occupies (`app.go:374-378`); the following argument arrives as a literal, and no value case in any parser can consume a literal. Before this phase both invocations stored the two-dash value — `ValidateFlags` skipped it as the value of a registered `TakesValue` flag, and `parseArgs` passed it through because `applyGlobalFlag` returns false for `--`. This phase's own tests create a task whose title is exactly `--` (`end_of_flags_test.go:159-172`), so the tool now produces a stored value it cannot write back: read `--` out of `tick show`, write it back with `tick update <id> --title --`, and the command errors. That is the §2.2 byte-identity bar failing on a value the tool itself accepts, and it is noticed as a flat refusal with no way round it — there is no escape spelling, since `--title -- --` fails identically (post-marker arguments become positionals and never re-attach to the flag). A description of exactly `--` has no positional route at all and is unwritable on both `create` and `update`. The awkward-task fixture carries no dash-only value (`round_trip_test.go:14-24`), so nothing in the suite fails.
- **Evidence**:
  - `internal/cli/app.go:374-378` — the marker is recognised at any position, including a flag's value slot
  - `internal/cli/flags.go:176-188` — `splitLiteralArgs` hands the post-marker argument to the parser as a literal
  - `internal/cli/update.go:53-64` (`--title`, `--description`), `internal/cli/create.go:56-60` (`--description`) — the "requires a value" returns that now fire
  - `internal/cli/create.go:39`, `internal/cli/update.go:46`, `internal/cli/show_fields.go:246`, `internal/cli/list.go:37` — every parser consumes values from `flagArgs` only
  - `internal/cli/end_of_flags_test.go:159-172` — the phase stores a task titled `--`
  - `internal/cli/round_trip_test.go:14-24` — the byte-identity fixture's values, none dash-only
- **Proposed shape**: support the attached form `--flag=value` for value-taking flags. `applyGlobalFlag` already leaves `--description=--` alone; `ValidateFlags` splits on the first `=` before the registry lookup (`flags.go:127-160`), and each parser's value cases accept the attached spelling alongside the separated one. That closes the identical pre-existing hole in the same move — a value that spells a global flag is stripped by `applyGlobalFlag` at any position today, so `--description --json` fails now and would keep failing under any marker-only fix — which is why it belongs at the phase boundary rather than in either parser. Extend the round-trip fixture with a dash-only and a flag-spelling value on both `--title` and `--description`. A smaller alternative exists — let a value-taking flag that runs out of `flagArgs` consume `literals[0]` — but it is ambiguous exactly where it matters: when the flag is the last `flagArg`, the first literal is equally the command's positional.
- **Bank**: `free-text-round-trip-5-1` (reviewer) — "A value-taking flag whose value is exactly `--` became unwritable, leaving a new hole in §2.2's byte-identity bar that no command can now close." Confirmed against the final state: no task in the phase added `--flag=value` support or any other route, and task 5-2's split does not reach the value slot.

## Comment Corrections

- `internal/cli/app.go:364-366` — "non-global arguments only" is false after this phase: a global flag placed after the marker is left in the returned args, which is the whole point of the marker (`app.go:375-395`, pinned by `end_of_flags_test.go:107-120`). The line is also the only one in the file past the wrap.
  OLD:
  ```
  // leaves the next argument as the subcommand and does not count it. Returns the parsed global flags, the subcommand name, remaining
  // subcommand-specific args (non-global arguments only), and an error if an
  // unknown flag appears before the subcommand.
  ```
  NEW:
  ```
  // leaves the next argument as the subcommand and does not count it. Returns the
  // parsed global flags, the subcommand name, the remaining subcommand-specific
  // args, and an error if an unknown flag appears before the subcommand.
  ```

- `internal/cli/flags.go:19-20` — "stripped by parseArgs before dispatch" no longer holds for every occurrence: `parseArgs` leaves a global flag that follows the marker in the args (`app.go:375-382`). The second clause names a path that does not exist either — `parseArgs` always runs before validation, so "validation runs before global stripping" describes nothing.
  OLD:
  ```
  // These are stripped by parseArgs before dispatch but may appear in subArgs
  // when validation runs before global stripping.
  ```
  NEW:
  ```
  // parseArgs applies and drops them wherever they appear before the end-of-flags
  // marker; after it they are ordinary text.
  ```

## Spec Defects

### S1: §10.2's "purely additive" claim is wrong — `--` was already accepted as a flag value

- **Claim**: §10.2, first bullet: "**`--` is supported as the end-of-flags marker, and becomes the canonical documented way to pass free text that may begin with a dash.** Purely additive: `--` is currently rejected on every command (`unknown flag "--" for "note add"`), so no existing invocation uses it. Everything that works today works identically, and inputs that currently fail begin to succeed."
- **Observed**: `--` was not rejected on every command before the change. As the value of a registered value-taking flag it was accepted and stored: `ValidateFlags` skipped the argument following a `TakesValue` flag (`git show b9782df7:internal/cli/flags.go`, the `def.TakesValue { i++ }` branch) and `parseArgs` passed it through, since `applyGlobalFlag` returns false for `--` (`app.go:427-448`). `tick update <id> --description --` therefore stored the description `--` before this phase and now fails with `--description requires a value` (F2). One class of invocation that worked does not work identically.
- **Read**: spec stale. The rule §10.2 sets is right and §2.2's bar is what F2 restores; the sentence asserting the change carries no regression is the wrong part, and it is wrong about the pre-change behaviour it measured, not about the design. Worth a corrigendum recording that the marker's introduction takes one case away — a value-taking flag whose value is exactly `--` — and that the fix is the attached `--flag=value` form rather than the marker alone.
