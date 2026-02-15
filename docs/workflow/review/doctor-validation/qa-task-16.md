TASK: Extract shared JSONL line iterator and parse tasks.jsonl once per doctor run

ACCEPTANCE CRITERIA:
- tasks.jsonl is opened at most twice per full doctor run (once for line-level scanning, once for relationship parsing)
- No line-level check contains file-open, bufio.Scanner, or line-counting boilerplate
- No relationship check calls ParseTaskRelationships directly in its Run method (it reads from context)
- All existing doctor tests pass without modification (or with minimal test setup changes)
- JsonlSyntaxCheck still correctly reports line numbers for malformed JSON

STATUS: Complete

SPEC CONTEXT: The doctor command runs 9 error checks and 1 warning check in a single invocation. Three checks operate on raw JSONL lines (syntax, duplicate ID, ID format) and six checks operate on parsed task relationships (orphaned parent, orphaned dependency, self-referential dep, dependency cycle, child blocked by parent, parent done with open children). All checks are diagnostic-only and read-only. This refactoring task addresses I/O duplication where the file was being opened and parsed ~10 times per doctor run.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_reader.go:14-66 -- JSONLine struct and ScanJSONLines function
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_reader.go:69-74 -- JSONLinesKey context key (unexported key type, exported key variable)
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_reader.go:78-83 -- getJSONLines helper with context lookup and fallback
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_reader.go:88-94 -- getTaskRelationships derives relationships from cached JSONLines
  - /Users/leeovery/Code/tick/internal/doctor/task_relationships.go:22-75 -- taskRelationshipsFromLines pure function (no I/O)
  - /Users/leeovery/Code/tick/internal/cli/doctor.go:30-35 -- RunDoctor pre-scans JSONL once and stores in context
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_syntax.go:20 -- uses getJSONLines instead of direct file access
  - /Users/leeovery/Code/tick/internal/doctor/duplicate_id.go:26 -- uses getJSONLines instead of direct file access
  - /Users/leeovery/Code/tick/internal/doctor/id_format.go:23 -- uses getJSONLines instead of direct file access
  - /Users/leeovery/Code/tick/internal/doctor/orphaned_parent.go:18 -- uses getTaskRelationships instead of ParseTaskRelationships
  - /Users/leeovery/Code/tick/internal/doctor/orphaned_dependency.go:18 -- uses getTaskRelationships
  - /Users/leeovery/Code/tick/internal/doctor/self_referential_dep.go:18 -- uses getTaskRelationships
  - /Users/leeovery/Code/tick/internal/doctor/dependency_cycle.go:21 -- uses getTaskRelationships
  - /Users/leeovery/Code/tick/internal/doctor/child_blocked_by_parent.go:22 -- uses getTaskRelationships
  - /Users/leeovery/Code/tick/internal/doctor/parent_done_open_children.go:21 -- uses getTaskRelationships
- Notes:
  - The implementation actually achieves better than the "at most twice" target. The file is scanned at most ONCE per doctor run: RunDoctor calls ScanJSONLines once and stores the result in context. Both line-level checks (via getJSONLines) and relationship checks (via getTaskRelationships -> getJSONLines -> taskRelationshipsFromLines) read from context. The original task_relationships.go ParseTaskRelationships now also delegates to ScanJSONLines internally, and getTaskRelationships derives relationships from the cached JSONLines without re-opening the file.
  - The approach evolved from the plan's "TaskRelationshipsKey context key" approach to a cleaner design: only JSONLinesKey is stored in context, and getTaskRelationships derives relationship data from the cached lines via the pure function taskRelationshipsFromLines. This is a positive deviation -- simpler, single source of truth.
  - bufio.Scanner and os.Open for tasks.jsonl exist only in jsonl_reader.go (the shared iterator).
  - No relationship check calls ParseTaskRelationships in its Run method (confirmed via grep).
  - CacheStalenessCheck still uses os.ReadFile separately for SHA256 hashing, which is correct -- it needs raw bytes, not parsed lines.

TESTS:
- Status: Adequate
- Coverage:
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_reader_test.go: TestScanJSONLines (7 subtests) covers missing file, empty file, blank line skipping with correct line numbers, valid JSON parsing, invalid JSON with nil Parsed and Raw populated, Raw populated for valid JSON, whitespace-only line skipping
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_reader_test.go: TestGetJSONLines (2 subtests) covers context-based retrieval and fallback to ScanJSONLines
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_reader_test.go: TestGetTaskRelationships (2 subtests) covers deriving from cached JSONLines and fallback
  - /Users/leeovery/Code/tick/internal/doctor/task_relationships_test.go: TestTaskRelationshipsFromLines (6 subtests) covers the pure conversion function
  - /Users/leeovery/Code/tick/internal/doctor/jsonl_syntax_test.go: Existing tests pass -- still verify line numbers for malformed JSON (Line 2, Line 3, Line 4 assertions present)
  - All existing check tests continue to work because they pass ctxWithTickDir (which is just context.Background()), triggering the fallback path in getJSONLines
- Notes: Test coverage is well-balanced. The fallback path (context key missing) is tested explicitly, and the context-populated path is also tested. Existing tests exercise the fallback path naturally since ctxWithTickDir does not set JSONLinesKey.

CODE QUALITY:
- Project conventions: Followed -- stdlib testing only, t.Run subtests, t.Helper on helpers, error wrapping with fmt.Errorf, functional approach
- SOLID principles: Good -- ScanJSONLines has single responsibility (file I/O + parsing), taskRelationshipsFromLines is a pure function (no I/O), getJSONLines/getTaskRelationships are small composable helpers. The context-based DI pattern follows the existing codebase pattern.
- Complexity: Low -- All new functions are straightforward linear scans with no complex branching
- Modern idioms: Yes -- Uses context.Value for DI (idiomatic Go pattern for request-scoped data), unexported context key type to prevent collisions, pure functions for data transformation
- Readability: Good -- Well-documented exported types and functions, clear naming (JSONLine, ScanJSONLines, getJSONLines, getTaskRelationships, taskRelationshipsFromLines), intent is obvious
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The ctxWithTickDir helper in cache_staleness_test.go takes a tickDir parameter but ignores it (uses _ receiver). This is not introduced by this task but is a pre-existing minor oddity. It returns a bare context.Background() which is correct for testing the fallback path.
- ParseTaskRelationships is still exported and used in its own test file. It could potentially be unexported if no external callers exist, but it serves as a useful public API for the package and its tests validate the end-to-end path (ScanJSONLines -> taskRelationshipsFromLines).
