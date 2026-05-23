# Report: Task 1-1 — Wire --version global flag

**Task ID**: tick-9c68fe
**STATUS**: Complete
**FINDINGS_COUNT**: 0

## Acceptance Criteria

- Add `version` bool to `globalFlags`.
- Recognise `--version` in `applyGlobalFlag`.
- Short-circuit in `App.Run` BEFORE empty-subcmd help branch and `ValidateFlags` / subcommand dispatch.
- Emit identical output to existing `version` subcommand (`tick version <Version>\n`).
- Existing `version` subcommand branch untouched (functionally).
- Tests cover both forms identical, no-subcommand case, precedence over other flags.

## Spec Context

Spec requests `--version` as an alias for the `version` subcommand, mirroring `--help` early dispatch. No short alias, no changes to `Version` sourcing. Output must be `tick version {Version}\n`.

## Implementation

- **Status**: Implemented (follow-up task 2-1 refactored inline printf into `printVersion` helper — verified against current state).
- **Locations** (all in `internal/cli/app.go`):
  - lines 18-20: `printVersion(w io.Writer)` helper centralising the format string.
  - lines 43-48: `--version` short-circuit in `App.Run`, BEFORE empty-subcmd help (line 50), version subcommand (line 56), help, doctor/migrate, format resolution, and `ValidateFlags`.
  - lines 56-59: existing `version` subcommand now also calls `printVersion(a.Stdout)` — behaviourally identical to prior inline printf.
  - line 343: `version bool` field added to `globalFlags`.
  - lines 418-419: `applyGlobalFlag` recognises `"--version"` and sets `flags.version = true`.
- **Notes**: Short-circuit ordering correctly precedes all dispatch/validation, satisfying the "combining with --json still prints version" edge case. Centralising via `printVersion` makes byte-for-byte identity true by construction.

## Tests

- **Status**: Adequate.
- **Location**: `internal/cli/cli_test.go:516-592` (`TestVersionFlag`, 3 subtests).
- **Coverage**:
  1. line 517: byte-equality between `tick --version` and `tick version` stdout, plus exact expected string check.
  2. line 554: `tick --version --json` short-circuits before `ValidateFlags` / format machinery.
  3. line 575: `tick --version` with no subcommand prints version, doesn't fall through to `printTopLevelHelp`.
- Not over-tested (no redundant assertions). Not under-tested (all three plan edge cases covered). Would fail if branches reordered or format string drifted in one place.

## Code Quality

- Project conventions: Followed (stdlib `testing`, `t.Run` "it does X" naming, `t.TempDir()`, idiomatic `fmt.Fprintf`).
- SOLID: Good — `printVersion` is single-responsibility; no new coupling.
- Complexity: Low — +1 bool field, +1 switch case, +1 if-branch, +3-line helper.
- Modern idioms: Yes.
- Readability: Good — comments on app.go:15-17 and 43-44 explain intent.
- Issues: None.

## Blocking Issues

None.

## Non-Blocking Notes

- [idea] `--version` is not listed in `printTopLevelHelp` global flags output — users seeing `--help` won't discover it. Worth a follow-up.
- [idea] No short alias `-V` added (per spec exclusion). Trivial single-line addition to `applyGlobalFlag` if requested later.
