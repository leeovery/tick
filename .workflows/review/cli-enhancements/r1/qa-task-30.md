TASK: cli-enhancements-6-1 -- Resolve partial IDs for list --parent

ACCEPTANCE CRITERIA:
- The --parent flag on list/ready/blocked commands resolves partial IDs through ResolveID before filtering

STATUS: Complete

SPEC CONTEXT: The specification states that partial ID matching "applies everywhere an ID is accepted: positional args, --parent, --blocked-by, --blocks." The --parent flag on list/ready/blocked is explicitly included in the scope of partial ID resolution. Resolution rules: strip tick- prefix, lowercase, min 3 hex chars, ambiguous prefix errors listing matches, zero matches returns not found.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/cli/list.go:156-161
- Notes: In `RunList()`, before any query work, the parent filter value is resolved via `store.ResolveID(filter.Parent)`. This is the single call path for all three commands: `list`, `ready` (via `handleReady` in app.go:179-188), and `blocked` (via `handleBlocked` in app.go:192-201). Both `handleReady` and `handleBlocked` prepend `--ready`/`--blocked` to their args, parse via `parseListFlags`, then delegate to `RunList`. The `parseListFlags` function at list.go:66 also calls `task.NormalizeID()` (lowercase) on the raw value, which is redundant but harmless since `ResolveID` does its own lowercasing. No drift from plan.

TESTS:
- Status: Adequate
- Coverage: Three dedicated subtests in parent_scope_test.go cover the core partial ID scenarios for list --parent:
  - Line 325: "it resolves partial parent ID and returns children" -- happy path with partial "a3f" resolving to "tick-a3f1b2", verifying children appear and unrelated tasks do not
  - Line 350: "it errors with ambiguous partial parent ID" -- two tasks sharing prefix "a3f", verifies error message contains "ambiguous" and both matching IDs
  - Line 374: "it errors with not found for non-matching partial parent ID" -- partial "zzz" with no match, verifies "not found" in stderr
  Additionally, the `ready --parent` and `blocked --parent` tests (lines 114-170) use full IDs but exercise the same `RunList` code path that includes `ResolveID`. Since `ResolveID` is tested at the storage layer (resolve_id_test.go) and the list integration tests verify the wiring, coverage is appropriate without needing separate partial-ID tests for ready/blocked specifically.
- Notes: No over-testing concerns. Tests are focused and each verifies a distinct behavior.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, t.Run subtests, t.TempDir via helpers, error wrapping with %w.
- SOLID principles: Good. ResolveID is a single-responsibility method in the storage layer. The list command delegates to it cleanly. No violations.
- Complexity: Low. The ResolveID integration is 5 lines of straightforward if-check-and-resolve at the top of RunList.
- Modern idioms: Yes. Standard Go error handling pattern with early return.
- Readability: Good. The intent is clear -- resolve before query.
- Issues: Minor -- the `NormalizeID` call in `parseListFlags` (line 66) lowercases the parent value, and then `ResolveID` lowercases it again internally. This double-normalization is harmless but slightly redundant. Not blocking.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `task.NormalizeID()` call at list.go:66 is redundant since `ResolveID` performs its own lowercasing. Could be removed for clarity, but it matches the pattern used for other flags (status, type) that normalize during parsing, so it is consistent and acceptable.
