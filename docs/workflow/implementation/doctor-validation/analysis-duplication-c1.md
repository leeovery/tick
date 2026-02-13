AGENT: duplication
FINDINGS:
- FINDING: JSONL file scanning loop duplicated across three line-level checks
  SEVERITY: high
  FILES: internal/doctor/jsonl_syntax.go:23-65, internal/doctor/duplicate_id.go:30-85, internal/doctor/id_format.go:27-102
  DESCRIPTION: Three checks (JsonlSyntaxCheck, DuplicateIdCheck, IdFormatCheck) each independently implement the same JSONL file-scanning loop: open tasks.jsonl, handle file-not-found error with identical result, create bufio.Scanner, maintain lineNum counter, skip blank/whitespace-only lines via strings.TrimSpace, and json.Unmarshal each line into map[string]interface{}. The DuplicateIdCheck and IdFormatCheck additionally share the identical id-field extraction pattern (check "id" key exists, assert string type, handle missing/non-string). This is approximately 15-20 lines of identical scaffolding per check.
  RECOMMENDATION: Extract a shared JSONL line iterator (e.g., a function or type in task_relationships.go or a new jsonl_reader.go) that opens the file, scans lines, skips blanks, parses JSON, and yields (lineNum, parsedObject) tuples. Each check then only implements its validation logic. Alternatively, since ParseTaskRelationships already does this for relationship checks, consider generalizing it or adding a lower-level variant that the line-level checks can use.

- FINDING: "tasks.jsonl not found" error result repeated in 9 check files
  SEVERITY: medium
  FILES: internal/doctor/jsonl_syntax.go:29-35, internal/doctor/duplicate_id.go:36-42, internal/doctor/id_format.go:33-39, internal/doctor/orphaned_parent.go:22-29, internal/doctor/orphaned_dependency.go:22-29, internal/doctor/self_referential_dep.go:22-29, internal/doctor/dependency_cycle.go:24-31, internal/doctor/child_blocked_by_parent.go:26-33, internal/doctor/parent_done_open_children.go:24-31
  DESCRIPTION: Every check constructs a nearly identical CheckResult for the "tasks.jsonl not found" case with Details "tasks.jsonl not found" and Suggestion "Run tick init or verify .tick directory". Only the Name field varies. This is 6-8 lines repeated 9 times across 9 files. If the suggestion wording ever changes, all 9 must be updated in sync.
  RECOMMENDATION: Extract a helper function like fileNotFoundResult(checkName string) []CheckResult in the doctor package that returns the standard "not found" error result. Each check calls this instead of constructing the literal.

- FINDING: knownIDs map construction duplicated across relationship checks
  SEVERITY: low
  FILES: internal/doctor/orphaned_parent.go:31-34, internal/doctor/orphaned_dependency.go:31-34, internal/doctor/dependency_cycle.go:35-38
  DESCRIPTION: Three checks independently build the same knownIDs set (map[string]struct{}) from ParseTaskRelationships results with identical 3-line loop. This is a minor duplication (3 lines x 3 files) but represents a concept ("set of all known task IDs") that could be provided by the shared parser.
  RECOMMENDATION: Consider adding a KnownIDs() method on a parsed result type or returning it alongside the task slice from ParseTaskRelationships. Low priority given the small code size.

- FINDING: createCacheWithHash test helper duplicated between test files
  SEVERITY: medium
  FILES: internal/doctor/cache_staleness_test.go:35-50, internal/cli/doctor_test.go:55-69
  DESCRIPTION: The createCacheWithHash function in cache_staleness_test.go and createDoctorCache in doctor_test.go are near-identical implementations that create a SQLite cache.db with a metadata table and insert a jsonl_hash value. They differ only in function name. Both are test helpers that create the same fixture.
  RECOMMENDATION: Extract a shared test helper into a testutil file (e.g., internal/doctor/testhelpers_test.go is already used by other tests via setupTickDir/writeJSONL/ctxWithTickDir). The CLI test could import or duplicate from a single authoritative location. Since these are in different packages (doctor vs cli), the CLI test helper may need to remain separate, but the naming and implementation should be aligned. Alternatively, make the doctor package's helper exported for test use via an internal test helper package.

SUMMARY: The primary duplication is the JSONL file-scanning loop replicated across three line-level checks (jsonl_syntax, duplicate_id, id_format), each independently opening the file, scanning lines, skipping blanks, and parsing JSON. A secondary pattern is the "tasks.jsonl not found" error result constructed identically in all 9 check files. Both are high-value extraction candidates that would reduce maintenance burden and ensure consistent behavior.
