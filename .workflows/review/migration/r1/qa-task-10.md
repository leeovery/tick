TASK: Unknown Provider Available Listing (migration-2-5)

ACCEPTANCE CRITERIA:
- NewProvider returns *UnknownProviderError (not a plain error) for unrecognized names
- UnknownProviderError.Error() produces the spec-mandated multi-line format with provider name in quotes
- Available providers are listed alphabetically, each on its own line with "  - " prefix
- An empty line separates the error line from the "Available providers:" section
- AvailableProviders() function exists and returns a sorted slice of registered provider names
- The CLI stderr output matches the spec format: Error: Unknown provider "<name>" followed by available list
- NewProvider("beads") still returns a valid BeadsProvider (regression)
- Single-provider listing is tested and correct
- Multiple-provider listing is tested (via test-scoped registry entries) and alphabetically sorted
- UnknownProviderError is exported and supports errors.As type assertion
- All tests written and passing

STATUS: Complete

SPEC CONTEXT:
The specification requires that if --from specifies an unrecognized provider, the CLI exits immediately with an error listing available providers in the exact format:
```
Error: Unknown provider "xyz"

Available providers:
  - beads
```
Phase 1 (migration-1-5) deferred this to Phase 2, using a minimal error. This task upgrades the error to the full spec format.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/migrate/errors.go:12-36 -- UnknownProviderError struct and Error() method
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:15 -- providerNames var
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:19-29 -- newMigrateProvider returns *UnknownProviderError for unknown names
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:32-37 -- availableProviders() returns sorted list
  - /Users/leeovery/Code/tick/internal/cli/migrate.go:93 -- CLI prints "Error: %s\n" which produces the spec-mandated output
- Notes:
  - The plan described placing AvailableProviders() in internal/migrate/registry.go as an exported function. The actual implementation places it as unexported availableProviders() in internal/cli/migrate.go. This is a minor architectural deviation from the plan but functionally equivalent -- the function is only called within the CLI package to populate UnknownProviderError.Available.
  - The plan described modifying a NewProvider() function in internal/migrate/registry.go, but Phase 1 placed the provider resolution in the CLI layer as newMigrateProvider(). This task correctly enhances that same function. The deviation is inherited from Phase 1 and is not a regression from this task.
  - The Error() method correctly sorts the Available slice defensively (copies then sorts), so even if the caller passes unsorted names, the output is deterministic and alphabetical.
  - The CLI error printing path at line 93 produces: "Error: Unknown provider \"xyz\"\n\nAvailable providers:\n  - beads\n" which matches the spec format exactly.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/migrate/errors_test.go:8-100 -- TestUnknownProviderError covers:
    - Error includes unknown provider name in quotes (line 9)
    - Error includes Available providers header (line 21)
    - Error lists each provider with "  - " prefix (line 33)
    - Empty line between error line and available list (line 45)
    - Single provider in registry (line 59) -- edge case
    - Multiple providers alphabetically sorted (line 71) -- edge case, verifies out-of-order input is sorted
    - Type-assertable via errors.As (line 84)
  - /Users/leeovery/Code/tick/internal/cli/migrate_test.go:16-86 -- TestNewMigrateProvider covers:
    - Registry returns BeadsProvider for "beads" (line 17) -- regression
    - Returns UnknownProviderError for unrecognized name (line 33)
    - Regression: still returns BeadsProvider for "beads" (line 50)
    - availableProviders returns sorted list containing "beads" (line 63)
  - /Users/leeovery/Code/tick/internal/cli/migrate_test.go:128-147 -- CLI prints Error: Unknown provider followed by available providers to stderr
- Notes:
  - Tests for UnknownProviderError.Error() in errors_test.go lines 9-18 and 21-30 are essentially duplicates: both construct the error with the same Available (["beads"]) and assert the same output format, differing only in the Name field ("xyz" vs "jira"). The second test's name says "includes Available providers header" but verifies the entire string, not just the header. This is mild over-testing but not blocking -- the tests are fast and not complex.
  - The multiple-providers edge case test (line 71) correctly passes providers in non-alphabetical order (["linear", "beads", "jira"]) and verifies they appear sorted. This is well-designed.
  - The CLI integration test (line 128) verifies the full stderr output using substring checks, confirming the "Error: " prefix, the "Available providers:" header, and the "  - beads" listing all appear.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, error wrapping with fmt.Errorf, exported error type with doc comments
- SOLID principles: Good -- UnknownProviderError has single responsibility (formatting the error), the Error() method is self-contained, the Available field allows programmatic inspection
- Complexity: Low -- Error() method is straightforward: copy, sort, build string
- Modern idioms: Yes -- uses strings.Builder for efficient string construction, sort.Strings for deterministic output, defensive copy before sort to avoid mutating the caller's slice
- Readability: Good -- clear struct fields (Name, Available), well-documented Error() method with godoc showing the output format
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The plan specified AvailableProviders() as an exported function in internal/migrate/. The actual implementation is unexported in internal/cli/. If future code outside the CLI package needs to enumerate providers (e.g., shell completion, programmatic tools), it would need to be moved and exported. For now, this is fine since the only consumer is the CLI.
- Tests at errors_test.go lines 9-18 and 21-30 are near-duplicates. The second could be removed or repurposed to test a meaningfully different scenario (e.g., empty Available list) without losing coverage.
- Tests at errors_test.go lines 45-57 ("empty line between error line and available list") and lines 59-69 ("single provider") also produce identical want strings. The naming distinguishes intent, but the assertions are the same as the first test. Three tests verify the exact same output for the single-provider case.
