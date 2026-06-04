---
topic: version-flag-help-and-alias
cycle: 1
total_proposed: 2
---
# Analysis Tasks: Version Flag Help And Alias (Cycle 1)

## Task 1: Add --version and -V to globalFlagSet
status: pending
severity: low
sources: architecture, standards

**Problem**: `applyGlobalFlag` now recognises `--version`/`-V` as global flags, but `globalFlagSet` in `internal/cli/flags.go:20-30` (the registry `ValidateFlags` consults to treat an arg as a valid global flag for any command) omits both `--version` and the new `-V`, even though it lists every other global flag including `--help`/`-h`. The two sources of truth — the `applyGlobalFlag` switch and the `globalFlagSet` map — now disagree about what counts as a global flag, and the map's doc comment claims it contains "all global flags." Currently masked because `parseArgs` strips global flags before dispatch and `flags.version` short-circuits before `ValidateFlags` runs, but the omission becomes a real "unknown flag" bug for `-V` if validation ordering changes or `globalFlagSet` gains a second consumer. This is exactly the kind of registry drift the codebase guards against elsewhere (e.g. the `commandFlags` drift-detection test).

**Solution**: Add `"--version": true` and `"-V": true` to the `globalFlagSet` map in `internal/cli/flags.go`, aligning the registry with `applyGlobalFlag` and the map's doc comment. This subsumes the pre-existing `--version` gap.

**Outcome**: `globalFlagSet` lists every flag `applyGlobalFlag` recognises, including `--version` and `-V`; the doc comment is accurate again; and `ValidateFlags` would accept `--version`/`-V` if they ever reached it. No behaviour change for existing dispatch paths.

**Do**:
1. Open `internal/cli/flags.go` and locate `globalFlagSet` (~lines 20-30).
2. Add two entries: `"--version": true` and `"-V": true`, matching the existing entry style (e.g. alongside `"--help"`/`"-h"`).
3. Run `gofmt -w internal/cli/flags.go` and `go vet ./...`.

**Acceptance Criteria**:
- `globalFlagSet` contains `"--version": true` and `"-V": true`.
- No existing global flag entry is removed or altered.
- `go test ./...`, `go vet ./...`, and `gofmt -l internal/cli/` are clean.

**Tests**:
- If a drift-detection test or table covers `globalFlagSet` membership, extend it to assert `--version` and `-V` are present. Otherwise no new test is strictly required — existing CLI tests must continue to pass.

## Task 2: Collapse duplicated version-flag tests into a table-driven case
status: pending
severity: low
sources: duplication

**Problem**: In `internal/cli/cli_test.go`, the new `-V` short-alias subtest (~lines 554-589) is a near-verbatim ~36-line copy of the existing `--version` subtest (~lines 517-552). The two blocks are identical apart from the flag string (`-V` vs `--version`) and the `alias`/`flag` variable-name prefixes. Both assert the same contract: the flag's stdout equals the `version` subcommand's stdout, equals `"tick version " + Version + "\n"`, with empty stderr and exit 0. The parallel copies will drift if the version-output contract changes — any edit must be applied in both places.

**Solution**: Consolidate the two subtests into a single table-driven subtest iterating over the input args (`{"--version"}` and `{"-V"}`), each compared against the `{"version"}` subcommand output. Extract the repeated `App{Stdout/Stderr/Getwd}` construction, `Run`, and assertion sequence into the loop body (or a small helper) so the equality-to-subcommand and exact-output assertions live once.

**Outcome**: A single table-driven subtest covers both `--version` and `-V` with one copy of the setup-and-assert logic; the version-output contract is asserted in exactly one place; coverage of both flags is preserved.

**Do**:
1. Open `internal/cli/cli_test.go` and locate the `--version` subtest (~517-552) and the `-V` subtest (~554-589).
2. Replace both with one table-driven subtest: a slice of cases, each with a name and the flag args (`[]string{"--version"}`, `[]string{"-V"}`).
3. In the loop body, run the `version` subcommand once (or per case) to get the expected output, then run the flag args through the same `App` setup, and assert stdout equals the subcommand output and equals `"tick version " + Version + "\n"`, stderr is empty, and the call returns no error.
4. Preserve the existing test/subtest naming convention ("it does X").
5. Run `go test ./internal/cli/`, `gofmt -w internal/cli/cli_test.go`, and `go vet ./...`.

**Acceptance Criteria**:
- The two duplicated subtests are replaced by a single table-driven subtest covering both `--version` and `-V`.
- The setup-and-assert sequence (App construction, Run, equality-to-subcommand, exact-output, empty-stderr, no-error) appears once.
- Both `--version` and `-V` remain covered; assertions are unchanged in substance.
- `go test ./internal/cli/`, `go vet ./...`, and `gofmt -l internal/cli/` are clean.

**Tests**:
- The consolidated table-driven subtest itself is the test; it must pass for both `--version` and `-V` cases and assert byte-identical output to the `version` subcommand.
