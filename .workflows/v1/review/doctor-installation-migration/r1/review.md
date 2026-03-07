# Implementation Review: Doctor, Installation, Migration

**Scope**: Multi-Plan (doctor-validation, installation, migration)
**QA Verdict**: Comments Only

## Summary

All three features are thoroughly implemented and well-tested. 49 of 52 tasks are fully complete with zero issues. The remaining 3 tasks have minor plan compliance gaps — not functional defects. Two are consequences of a post-plan Homebrew tap migration to an external repository (a deliberate architectural improvement), and one is a test helper adoption gap (9/10 checks use the extracted helper, 1 retains inline logic). The product is robust, architecturally coherent, and ready for use.

## QA Verification

### Specification Compliance

All three specifications are faithfully implemented:

- **Doctor**: All 9 error checks and 1 warning check run in a single invocation. Human-readable output with markers, details, fix suggestions, correct exit codes. Doctor never modifies data — verified by read-only tests on every check.
- **Installation**: goreleaser produces archives for all 4 platforms with correct naming. GitHub Actions triggers on semver tags. Install script handles Linux (direct download) and macOS (Homebrew delegation). Homebrew formula migrated to external `homebrew-tools` repo with automated `repository_dispatch`.
- **Migration**: `tick migrate --from beads` imports tasks correctly. Provider contract, dry-run, pending-only, continue-on-error, failure reporting, and unknown provider listing all work as specified.

Notable spec drift (non-breaking):
- Install script uses `brew install leeovery/tools/tick` instead of `brew tap leeovery/tick && brew install tick` — this is simpler and correct for the multi-tool tap migration. An improvement over spec.

### Plan Completion

**Doctor-Validation** (6 phases, 22 tasks):
- [x] Phase 1 acceptance criteria met (framework, cache check, output, exit codes)
- [x] Phase 2 acceptance criteria met (JSONL syntax, ID format, duplicate ID)
- [x] Phase 3 acceptance criteria met (all relationship checks, warnings don't affect exit code)
- [x] Phases 4-6 analysis refactoring completed
- [x] All 22 tasks completed
- [x] No scope creep

**Installation** (5 phases, 15 tasks):
- [x] Phase 1 acceptance criteria met (binary, goreleaser, release workflow, Linux install)
- [x] Phase 2 acceptance criteria met (macOS paths, error hardening)
- [x] Phases 3-5 analysis refactoring completed
- [x] 14 of 15 tasks fully complete; 1 superseded by tap migration (installation-3-3)
- [x] No scope creep

**Migration** (4 phases, 15 tasks):
- [x] Phase 1 acceptance criteria met (provider contract, beads, engine, output, CLI)
- [x] Phase 2 acceptance criteria met (continue-on-error, failure output, dry-run, pending-only, unknown provider)
- [x] Phases 3-4 analysis refactoring completed
- [x] All 15 tasks completed
- [x] No scope creep

### Code Quality

No issues found. All three features follow consistent patterns:
- Clean separation of concerns (Check interface, Provider interface, TaskCreator interface)
- Dependency injection via struct fields and functional options
- Error wrapping with `fmt.Errorf("context: %w", err)` throughout
- Idiomatic Go: typed constants, map-based validation, table-driven tests

### Test Quality

Tests adequately verify requirements. Coverage is thorough across all three features with stdlib `testing` only, `t.Run()` subtests, `t.TempDir()` for isolation, and `t.Helper()` on helpers.

Minor gaps (non-blocking):
- `dry_run_creator_test.go:28` uses raw `"done"` string instead of `task.StatusDone` constant
- `cache_staleness_test.go` uses inline read-only verification instead of the shared `assertReadOnly` helper (9/10 checks use it)

### Required Changes

None. All findings are non-blocking.

## Product Assessment

### Robustness

- Doctor's CacheStalenessCheck reads tasks.jsonl independently (raw bytes for SHA256) rather than using the shared scanner — architecturally justified, produces slightly different error message format than other checks. Cosmetic, not a breakage.
- Beads provider handles malformed JSON via sentinel `Status: "(invalid)"` entries — creative but obscures root cause in error messages. Acceptable for one-time imports with small datasets.
- Migration inserts tasks individually (one Mutate per task) — correct but slow for large imports. Fine for v1 with beads as sole provider.
- Install script `parseMigrateArgs` silently ignores unknown flags — consistent with global parseArgs but could mask typos.

### Gaps

- **CLI help text missing doctor and migrate** — `printUsage()` in `app.go:203-208` only lists `init`. Users cannot discover these commands from `tick` or `tick help`. This is the highest-priority gap.
- Homebrew formula content cannot be verified from this repo (lives in external `homebrew-tools`). The naming contract test validates 2 of 3 sources (goreleaser + install script, not formula).
- CLAUDE.md refers to "MD5 hash comparison" but code uses SHA256 throughout — documentation is stale.

### Cross-Plan Consistency

Strong. All three features follow identical patterns:
- Same Go conventions, testing patterns, error handling
- Same CLI handler signature pattern (`Run<Command>`)
- Both doctor and migrate bypass the format/formatter machinery — clean and consistent for human-facing utility commands
- Error output uses the same `fmt.Fprintf(a.Stderr, "Error: ...")` pattern

### Integration Seams

Clean integration across all three features:
- Migration writes through the same `openStore` helper as core commands — proper JSONL+SQLite dual-write with locking
- Doctor validates the same data structures and hashing algorithm (SHA256) as the storage layer
- Installation packages the same binary that all commands use
- Naming contract test bridges goreleaser config and install script, preventing asset naming drift

### Strengthening Opportunities

1. **(High)** Add `doctor` and `migrate` to CLI help/usage output
2. **(Medium)** Consider a `tick version` command — useful for debugging now that the release pipeline exists
3. **(Medium)** Add comment in naming contract test documenting why Homebrew formula leg is absent (external repo)
4. **(Low)** Update CLAUDE.md "MD5 hash comparison" → "SHA256 hash comparison"
5. **(Low)** Extend `assertReadOnly` helper to accept optional additional file paths, then use it in `cache_staleness_test.go`

### What's Next

These three features complete significant capability:
- **Doctor** enables data integrity debugging — a safety net for production use
- **Installation** enables distribution — users can now install tick
- **Migration** enables adoption — users can import existing task data

Natural next steps:
- Add remaining core commands to CLI help text
- Future migration providers (JIRA, Linear) when needed — the provider architecture supports it
- Consider `tick version` command for user support

## Recommendations

1. Add doctor and migrate to `printUsage()` — quick fix, high impact on discoverability
2. Update CLAUDE.md hash algorithm reference
3. Accept Homebrew formula tasks (installation-2-1, 3-3, 3-4) as superseded by tap migration — update plan task notes accordingly
4. Fix `dry_run_creator_test.go:28` to use `task.StatusDone` for consistency
5. Consider extending `assertReadOnly` helper for cache staleness special case
