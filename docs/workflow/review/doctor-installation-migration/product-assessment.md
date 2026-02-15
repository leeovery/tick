SCOPE: multi-plan
PLANS_REVIEWED: doctor-validation, installation, migration

ROBUSTNESS:
- Doctor CacheStalenessCheck (internal/doctor/cache_staleness.go:22-81) reads tasks.jsonl independently via os.ReadFile rather than using the shared ScanJSONLines/context caching path. This is architecturally justified -- it needs raw bytes for SHA256 hashing, not parsed structures -- but means a file-read failure in the cache check produces a different error message format ("tasks.jsonl not found or unreadable") than the shared fileNotFoundResult helper produces ("tasks.jsonl not found"). Under real use this is cosmetic, not a breakage.

- The beads provider (internal/migrate/beads/beads.go:92-101) handles malformed JSON by creating a sentinel entry with Status: "(invalid)" which is then rejected by the engine Validate() call. This is a creative approach that keeps the provider simple, but the error message seen by users will be 'invalid status "(invalid)"' rather than something like "malformed JSON on line N". The user loses the root cause. For a single provider used in one-time imports, this is acceptable but would become confusing with larger datasets.

- The install script (scripts/install.sh:155-156) creates a temp directory and sets a trap for cleanup, which is solid. However, if TICK_TEST_TARBALL is set but the file does not exist, the cp on line 163 will fail with a confusing "No such file or directory" error. This only affects test scenarios, not production use.

- Migration StoreTaskCreator.CreateTask (internal/migrate/store_creator.go:35-90) calls store.Mutate for each task individually. For large imports (hundreds of tasks), this means hundreds of individual lock-acquire/write/release cycles against the JSONL+SQLite store. This will be slow but correct. For v1 with beads as the sole provider (typically dozens of tasks), this is fine.

- The parseMigrateArgs function (internal/cli/migrate.go:48-68) silently ignores unknown flags. Running "tick migrate --from beads --unknown-flag" will succeed without warning. This is consistent with how the global parseArgs works but could mask typos in flag names (e.g., --dry_run instead of --dry-run).

GAPS:
- No tick migrate or tick doctor entry in the CLI usage/help text. The printUsage() function in app.go:203-208 only lists init. Running tick or tick help gives no indication that doctor or migrate exist. This is a real user-facing gap -- someone who installs tick and runs it without arguments will not discover these commands.

- The install script spec (installation specification) describes macOS Homebrew as "brew tap {owner}/tick && brew install tick", but the implementation uses "brew install leeovery/tools/tick" (scripts/install.sh:66). The implementation is simpler and correct for the new homebrew-tools multi-tool tap. The error message on line 62 similarly says "brew install leeovery/tools/tick" rather than the two-step tap+install. This is an improvement over the spec, not a deficiency, but the spec is now outdated.

- Migration has no mechanism to detect or warn about duplicate imports. The spec explicitly states "User responsibility" for duplicates from re-running, and this is documented. However, there is no --force flag or confirmation prompt -- re-running silently creates duplicates. This is a deliberate design choice, not an oversight.

INTEGRATION:
- Doctor and migration both bypass the format/formatter machinery in app.go:30-35, which is a clean and consistent pattern. Both are human-facing diagnostic/utility commands that should not produce TOON/JSON output.

- The hashing algorithm is consistent: internal/storage/cache.go uses SHA256, and internal/doctor/cache_staleness.go independently implements SHA256. Both query the same metadata table with key jsonl_hash. The doctor check correctly mirrors the storage layer hash mechanism. Note: CLAUDE.md refers to "MD5 hash comparison" which is outdated -- the actual code uses SHA256 throughout.

- The openStore function used by migration (internal/cli/migrate.go:113) is the same shared helper used by other commands (init, create, list, etc.), ensuring migration writes go through the same JSONL+SQLite dual-write path with proper locking.

- The naming contract test (scripts/naming_contract_test.go) bridges the installation and release concerns by verifying goreleaser config and install script produce identical asset filenames. This is a strong cross-component integration test.

CONSISTENCY:
- Error output patterns: Doctor writes to stdout (the diagnostic report itself), while migration uses stdout for progress and the CLI layer writes errors to stderr. Both doctor and migrate CLI handlers follow the same fmt.Fprintf(a.Stderr, "Error: ...") pattern for pre-command errors. This is consistent.

- Error propagation models differ by design: Doctor uses CheckResult with Severity and aggregates all results into a DiagnosticReport. Migration uses Result with Success/Err and aggregates into a slice. These are different types because they serve different purposes -- doctor checks are multi-valued per check (one check can return N results), while migration is one result per task. The difference is appropriate.

- Both doctor and migration define their own "file not found" error handling inline rather than sharing a common utility, but this is appropriate since the packages are independent and their error messages need different context.

- Test patterns are consistent across all three plans: table-driven tests with t.Run(), t.TempDir() for isolation, t.Helper() on helpers, stdlib testing only (no testify). This matches the project conventions in .claude/skills/golang-pro/SKILL.md.

- The installation tests use a novel pattern of testing shell scripts from Go (scripts/install_test.go) by invoking bash with environment variable injection. This works well and is thoroughly tested.

STRENGTHENING:
- The printUsage() function should list all available commands including doctor and migrate. This is the highest-priority improvement since it directly affects discoverability.

- Consider adding a --verbose or --debug flag to tick doctor that shows which checks are running, the JSONL file path, and the hash values being compared. Currently all 10 checks run silently until the report -- if a check is slow, there is no progress indication.

- The migration per-task Mutate pattern could be optimized with a batch insertion path if performance becomes a concern with future providers that import larger datasets. This is not urgent for v1.

NEXT_STEPS:
- (High) Add doctor and migrate to CLI help/usage output so users can discover them
- (Medium) Consider a tick version command now that the release pipeline and goreleaser are in place -- useful for debugging and support
- (Medium) Add Linux arm64 testing to CI if the project targets that platform (goreleaser builds it but nothing verifies it)
- (Low) The CLAUDE.md project documentation refers to "MD5 hash comparison" but the code uses SHA256 -- update the documentation to match reality
- (Low) Future migration providers (JIRA, Linear mentioned in spec) would benefit from a provider registration pattern rather than the current switch statement in newMigrateProvider, but this is premature optimization until a second provider is needed

SUMMARY: The three features form a coherent, well-tested addition to the tick product. Doctor provides comprehensive data integrity validation with 10 checks, clean separation of concerns via the Check interface, and efficient single-scan caching via context. Installation delivers a complete release pipeline from goreleaser through GitHub Actions to platform-specific install paths, with a particularly strong naming contract test that prevents drift between components. Migration implements a clean plugin architecture with proper DI via TaskCreator, continue-on-error semantics, and dry-run support.

Cross-plan consistency is strong: all three features follow the same Go conventions, testing patterns, and error handling approaches. The integration seams are clean -- migration writes through the same store as core commands, doctor validates the same data structures, and installation packages the same binary. The most notable gap is the missing CLI help entries for doctor and migrate, which undermines discoverability. Overall the product is robust and ready for use, with no blocking issues identified.
