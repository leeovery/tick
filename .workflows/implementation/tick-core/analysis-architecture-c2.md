AGENT: architecture
FINDINGS:
- FINDING: Ready query SQL duplicated between list.go and stats.go
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/list.go:209-224, /Users/leeovery/Code/tick/internal/cli/stats.go:86-99
  DESCRIPTION: The "ready" SQL WHERE conditions are independently authored in two locations. In list.go (buildListQuery, lines 209-224), the ready filter is three string conditions appended to a WHERE clause. In stats.go (RunStats, lines 86-99), the same logic is a standalone SQL query string. These are structurally identical but live in separate files with no shared constant or function. The code-quality standard ("Compose, Don't Duplicate") specifically calls out this pattern: "If you have a query for 'ready items,' the query for 'blocked items' should be derived." The stats.go correctly derives blocked from ready (line 105: Blocked = Open - Ready), but the ready SQL itself has no single source. If a new condition is added to the ready definition (e.g., a new field that affects readiness), it must be updated in both places independently. The list.go blocked query (lines 227-245) is the De Morgan inverse of the ready conditions, making three total places where ready semantics are encoded.
  RECOMMENDATION: Extract the ready WHERE clause fragments into a shared constant or function (e.g., readyConditions() returning []string) in a common location like format.go or a new query_helpers.go. The blocked conditions can then be derived as the negation. Both buildListQuery and RunStats would reference the same source. This confines the ready/blocked definition to one place.

- FINDING: Dead code: VerboseLog function in format.go
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/format.go:186-192
  DESCRIPTION: The standalone function VerboseLog(w io.Writer, verbose bool, msg string) is defined and tested (format_test.go:381-400) but never called in any production code path. All production verbose logging uses the VerboseLogger struct (verbose.go) instead. This function appears to be a remnant from before the VerboseLogger type was introduced.
  RECOMMENDATION: Remove VerboseLog from format.go and its associated tests from format_test.go. All verbose logging already flows through VerboseLogger.

SUMMARY: All cycle 1 findings have been addressed. The remaining architectural concern is the ready query SQL duplicated between list.go and stats.go, which creates three independent locations encoding ready/blocked semantics. The dead VerboseLog function is minor cleanup. Overall the architecture is clean with good separation between task model, storage orchestration, and CLI layers.
