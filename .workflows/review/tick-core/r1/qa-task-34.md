TASK: Extract store-opening boilerplate into shared helper (tick-core-7-3)

ACCEPTANCE CRITERIA:
- No inline DiscoverTickDir + NewStore sequence remains in any Run* function
- All 9 call sites use the shared openStore helper
- Each call site still has its own defer store.Close()
- All existing tests pass unchanged

STATUS: Complete

SPEC CONTEXT: The spec defines a dual-storage architecture (JSONL source of truth + SQLite cache) where every read/write operation must discover the .tick directory, open a storage.Store, and close it after use. This boilerplate was repeated identically across 9 call sites in 8 files (dep.go has two: RunDepAdd and RunDepRm). The task is a pure mechanical refactoring -- no behavioral changes expected.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/helpers.go:32-40 (openStore helper definition)
  - /Users/leeovery/Code/tick/internal/cli/create.go:103 (call site 1)
  - /Users/leeovery/Code/tick/internal/cli/dep.go:65 (call site 2 - RunDepAdd)
  - /Users/leeovery/Code/tick/internal/cli/dep.go:133 (call site 3 - RunDepRm)
  - /Users/leeovery/Code/tick/internal/cli/list.go:89 (call site 4)
  - /Users/leeovery/Code/tick/internal/cli/rebuild.go:12 (call site 5)
  - /Users/leeovery/Code/tick/internal/cli/show.go:38 (call site 6)
  - /Users/leeovery/Code/tick/internal/cli/stats.go:16 (call site 7)
  - /Users/leeovery/Code/tick/internal/cli/transition.go:20 (call site 8)
  - /Users/leeovery/Code/tick/internal/cli/update.go:122 (call site 9)
- Notes:
  - storage.NewStore appears only once in the CLI package: inside openStore at helpers.go:39
  - DiscoverTickDir is only called in helpers.go:35 (inside openStore), discover.go (definition), doctor.go:54 (doctor is a special case that does not use storage.NewStore), and test files
  - No remaining inline DiscoverTickDir + NewStore sequence in any Run* function
  - All 9 call sites verified to have their own defer store.Close()
  - migrate.go:113 also uses openStore (bonus consumer, not in original scope but consistent)

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/cli/helpers_test.go:292-340 -- TestOpenStore with 3 subtests:
    1. "it returns a valid store for a valid tick directory" -- happy path
    2. "it returns error when no tick directory exists" -- error case, checks error message contains "no .tick directory found"
    3. "it discovers tick directory from subdirectory" -- integration with DiscoverTickDir walk-up behavior
  - Existing command-level tests (create_test.go, dep_test.go, list_show_test.go, etc.) exercise openStore indirectly as part of full command flows
- Notes:
  - The three direct tests cover the task's test requirements exactly: valid store, error for missing .tick, and subdirectory discovery
  - The test for the error case correctly validates the error message content
  - Existing integration and command tests serve as regression coverage confirming no behavioral change
  - Not over-tested: tests are focused and minimal, no redundant assertions

CODE QUALITY:
- Project conventions: Followed. Helper placed in helpers.go alongside other shared helpers (outputMutationResult, parseCommaSeparatedIDs, applyBlocks). Unexported function -- correct for internal package helper.
- SOLID principles: Good. Single responsibility (openStore does one thing: discover + open). DRY achieved -- 9 identical sequences collapsed to 1.
- Complexity: Low. The function is a simple 2-step sequence: discover tick dir, open store. No branching beyond error checks.
- Modern idioms: Yes. Idiomatic Go error propagation. Variadic forwarding with storeOpts(fc)... is clean. Comment documents the defer requirement clearly.
- Readability: Good. Function name is self-documenting. Doc comment at line 32-33 clearly explains the caller's responsibility to defer store.Close().
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- doctor.go:54 still calls DiscoverTickDir directly (not through openStore) because it does not use storage.NewStore at all -- it passes the tickDir to the doctor package. This is intentional and correct, not a gap in the refactoring.
