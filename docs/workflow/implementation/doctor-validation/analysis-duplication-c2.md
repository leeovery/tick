AGENT: duplication
FINDINGS:
- FINDING: knownIDs map construction duplicated across three relationship checks
  SEVERITY: low
  FILES: internal/doctor/orphaned_parent.go:23-26, internal/doctor/orphaned_dependency.go:23-26, internal/doctor/dependency_cycle.go:27-30
  DESCRIPTION: Three checks independently build the same knownIDs set (map[string]struct{}) from the task relationship slice with an identical 3-line loop. This was flagged in cycle 1 as low severity and remains unaddressed. At 3 lines per occurrence, the total duplication is 9 lines.
  RECOMMENDATION: Low priority. Could add a buildKnownIDs helper or return the set alongside ParseTaskRelationships, but the code is trivial enough that extraction may be premature abstraction per the Rule of Three threshold for non-trivial blocks.

- FINDING: Read-only verification test pattern duplicated across 10 test files
  SEVERITY: medium
  FILES: internal/doctor/orphaned_parent_test.go:325-346, internal/doctor/orphaned_dependency_test.go:375-396, internal/doctor/self_referential_dep_test.go:314-335, internal/doctor/child_blocked_by_parent_test.go:387-409, internal/doctor/parent_done_open_children_test.go:464-486, internal/doctor/dependency_cycle_test.go:489-511, internal/doctor/jsonl_syntax_test.go:343-365, internal/doctor/duplicate_id_test.go:353-375, internal/doctor/id_format_test.go:446-468, internal/doctor/task_relationships_test.go:230-251
  DESCRIPTION: Every check test file contains a near-identical "does not modify tasks.jsonl (read-only verification)" test that follows the same pattern: write content, read file bytes before, run check, read file bytes after, compare. The logic is 12-15 lines per file across 10 files, totaling ~130 lines of duplicated scaffolding. Only the check type instantiated and the JSONL content vary.
  RECOMMENDATION: Extract a generic helper like assertReadOnly(t, tickDir, content []byte, runCheck func()) that handles the before/after file comparison. Each test then calls this with a one-line closure. This would reduce ~130 lines to ~10 lines of helper plus ~10 one-line calls.

- FINDING: "uses Name X for all results" table-driven test pattern duplicated across 9 test files
  SEVERITY: low
  FILES: internal/doctor/cache_staleness_test.go:223-276, internal/doctor/jsonl_syntax_test.go:207-249, internal/doctor/duplicate_id_test.go:250-292, internal/doctor/id_format_test.go:314-356, internal/doctor/orphaned_parent_test.go:224-266, internal/doctor/orphaned_dependency_test.go:275-317, internal/doctor/self_referential_dep_test.go:199-241, internal/doctor/dependency_cycle_test.go:402-446, internal/doctor/child_blocked_by_parent_test.go:282-326
  DESCRIPTION: Nine test files each contain a table-driven test verifying that all results use the correct Name field across three scenarios (passing, failing, missing file). The pattern is identical except for the check type and expected name string. Each block is ~25 lines, totaling ~225 lines. However, these are intentional per-check contract tests and extracting them would reduce test readability.
  RECOMMENDATION: Low priority. Could extract an assertCheckName(t, checkFactory, expectedName) helper, but the readability cost of abstracting test intent may outweigh the DRY benefit for test code.

- FINDING: createCacheWithHash test helper duplicated between doctor and cli packages
  SEVERITY: low
  FILES: internal/doctor/cache_staleness_test.go:35-50, internal/cli/doctor_test.go:55-69
  DESCRIPTION: The createCacheWithHash (doctor pkg) and createDoctorCache (cli pkg) functions are near-identical implementations that create a SQLite cache.db with a metadata table and insert a jsonl_hash value. They differ only in function name. This was noted in cycle 1 but remains because the helpers are in different test packages and Go does not allow cross-package test helper sharing without an exported test helper package.
  RECOMMENDATION: Low priority. The Go package boundary makes extraction non-trivial. An internal/testutil package could be created but may be over-engineering for two ~12-line helpers. Accept the duplication or align naming for easier future consolidation.

SUMMARY: Cycle 1 addressed the high-impact duplication (shared JSONL scanning, fileNotFoundResult helper, pre-parsed task relationships). The remaining duplication is primarily in test scaffolding -- the read-only verification pattern (medium, ~130 lines across 10 files) is the most impactful extraction candidate. The production code duplication (knownIDs map, 9 lines total) is below the extraction threshold.
