# Plan: Unknown Flags Silently Ignored

### Phase 1: Flag Validation and Normalizations
status: approved
ext_id: tick-fd039e
approved_at: 2026-03-10

**Goal**: All commands reject unknown flags with the specified error message. Flag metadata exported per command, validated centrally before dispatch. Prerequisite normalizations (dep rm to dep remove, --from= syntax removal) applied. Both dispatch paths (main switch and doctor/migrate) covered.

**Why this order**: This is the core fix. It starts with a failing test reproducing the bug, then builds the validation mechanism and wires it in. Normalizations are included because they are prerequisites that simplify the validator -- without them, the validator needs special cases for dep rm and --from=value.

**Acceptance**:
- [ ] Unknown flags on any command produce the error: `Error: unknown flag "{flag}" for "{command}". Run 'tick help {command}' for usage.`
- [ ] Two-level commands use fully-qualified name in error (e.g., "dep add") but parent in help reference (e.g., `tick help dep`)
- [ ] Global flags (--quiet, --verbose, --toon, --pretty, --json, --help/-h/-q/-v) are not rejected by command-level validation
- [ ] Unknown flags before the subcommand produce: `Error: unknown flag "{flag}". Run 'tick help' for usage.`
- [ ] `dep rm` is renamed to `dep remove`; `dep rm` returns an unknown sub-command error
- [ ] `--from=value` syntax is removed from `parseMigrateArgs`; only `--from value` (space-separated) works
- [ ] Both dispatch paths (main switch and doctor/migrate) validate flags before invoking handlers
- [ ] `version` and `help` commands are excluded from flag validation
- [ ] The specific `dep add --blocks` scenario from the bug report is tested and rejected
- [ ] Value-taking flags (e.g., `--priority 3`) correctly skip the value argument during validation

#### Tasks
| ID | Name | Edge Cases | Status | Ext ID |
|----|------|------------|--------|--------|
| unknown-flags-silently-ignored-1-1 | Normalize dep rm to dep remove and remove --from=value syntax | existing tests referencing dep rm, --from= with empty value | authored | tick-928bf7 |
| unknown-flags-silently-ignored-1-2 | Reproduce bug and build flag metadata with central validator | value-taking flags consuming next arg, short aliases (-f), two-level command error format | pending | |
| unknown-flags-silently-ignored-1-3 | Wire validation into parseArgs and both dispatch paths | pre-subcommand unknown flag error format, help/version bypass, doctor/migrate dispatch path | pending | |
| unknown-flags-silently-ignored-1-4 | Validate global flag pass-through and value-taking flag skipping | global flags interspersed with command args, --ready/--blocked on ready/blocked commands | pending | |

### Phase 2: Parser Cleanup and Regression Verification
status: approved
ext_id:
approved_at: 2026-03-10

**Goal**: Remove now-dead silent-skip logic from individual command parsers. Verify no regressions across the full command surface with dedicated regression tests.

**Why this order**: The `strings.HasPrefix(arg, "-")` skip logic in individual parsers is dead code after Phase 1 wires central validation. Removing it is a cleanup pass that must happen after the fix is proven working. Dedicated regression tests confirm nothing breaks when the per-parser skip logic is removed.

**Acceptance**:
- [ ] No `strings.HasPrefix(arg, "-")` skip logic remains in `parseCreateArgs`, `parseUpdateArgs`, `parseDepArgs`, `RunNoteAdd`, or `parseRemoveArgs`
- [ ] All existing tests pass with no regressions after parser cleanup
- [ ] Short flags (`-x`) are rejected as well as long flags (`--unknown`) across all commands
- [ ] Commands with no accepted flags (init, show, start, done, cancel, reopen, stats, doctor, rebuild, dep add, dep remove, note add, note remove) reject any flag passed to them
