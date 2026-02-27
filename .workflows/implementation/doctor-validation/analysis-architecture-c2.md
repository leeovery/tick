AGENT: architecture
FINDINGS:
- FINDING: ParseTaskRelationships duplicates ScanJSONLines parsing logic instead of deriving from it
  SEVERITY: medium
  FILES: internal/doctor/task_relationships.go:32, internal/doctor/jsonl_reader.go:27
  DESCRIPTION: ParseTaskRelationships independently opens tasks.jsonl, scans with bufio.Scanner, parses each line with json.Unmarshal into map[string]interface{}, skips blanks, and tracks line numbers -- all logic that ScanJSONLines already performs. The context-based caching in RunDoctor (internal/cli/doctor.go:32-39) prevents redundant I/O at runtime, but the two functions are structurally duplicate parsing pipelines. Per code-quality.md "Compose, Don't Duplicate": when new behavior is a subset of existing behavior, derive it from the existing abstraction. ParseTaskRelationships is a strict transformation of the JSONLine slice that ScanJSONLines produces. Having two independent parsers means a change to line-skipping rules, blank-line handling, or JSON decoding must be applied in both places or they drift.
  RECOMMENDATION: Refactor ParseTaskRelationships to accept []JSONLine as input (or add an internal variant) and transform it into []TaskRelationshipData. The file-reading version can be a thin wrapper that calls ScanJSONLines then transforms. This also simplifies the context-caching story in RunDoctor -- only JSONLines need to be cached; TaskRelationshipData is derived.

- FINDING: context.Value used as untyped service locator for pre-computed data
  SEVERITY: low
  FILES: internal/doctor/jsonl_reader.go:78-83, internal/doctor/jsonl_reader.go:95-100, internal/cli/doctor.go:32-39
  DESCRIPTION: Pre-scanned JSONLine data and TaskRelationshipData are passed to checks via context.WithValue with exported sentinel keys (JSONLinesKey, TaskRelationshipsKey). Each check retrieves this data with a type assertion inside getJSONLines/getTaskRelationships, falling back to re-reading the file if the key is absent. This makes the data flow implicit -- the Check interface signature (Run(ctx, tickDir)) gives no indication that context may carry pre-computed data. The fallback means correctness is not at risk, but the pattern obscures the actual dependency graph and means the compiler cannot verify that the context is populated correctly. Code-quality.md flags "untyped parameters when concrete types are known at design time."
  RECOMMENDATION: This is a pragmatic choice that avoids changing the Check interface. If this pattern grows (more context keys for more pre-computed data), consider a RunContext struct passed alongside or instead of context.Context. For now, the current approach is workable but worth noting as a watch item.

- FINDING: FormatReport computes its own issue count independently of DiagnosticReport methods
  SEVERITY: low
  FILES: internal/doctor/format.go:12-37, internal/doctor/doctor.go:60-80
  DESCRIPTION: FormatReport maintains a local issueCount variable that counts all non-passing results (errors + warnings). DiagnosticReport already provides ErrorCount() and WarningCount() methods that partition failures by severity. The format function's count is logically ErrorCount() + WarningCount(), but it is computed via a separate loop. Per "Compose, Don't Duplicate", the format function should derive its count from the report's existing methods rather than counting independently. If the definition of "issue" changes (e.g., a new severity level is added), FormatReport's count and the report methods could produce inconsistent numbers.
  RECOMMENDATION: Replace the local issueCount in FormatReport with report.ErrorCount() + report.WarningCount(), or add an IssueCount() method to DiagnosticReport that FormatReport calls. This keeps the count definition in one place.

SUMMARY: The main structural concern is ParseTaskRelationships duplicating ScanJSONLines' file-parsing logic rather than transforming its output -- a compose-don't-duplicate violation. Two lower-severity items (context-as-service-locator, parallel issue counting) are worth addressing but do not create correctness risk.
