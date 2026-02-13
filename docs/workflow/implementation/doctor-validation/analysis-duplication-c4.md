AGENT: duplication
FINDINGS:
- FINDING: createCacheWithHash duplicated across package boundaries
  SEVERITY: medium
  FILES: internal/doctor/cache_staleness_test.go:35, internal/cli/doctor_test.go:55
  DESCRIPTION: createCacheWithHash (doctor package, lines 35-50) and createDoctorCache (cli package, lines 55-69) are functionally identical -- same SQL statements, same structure, same purpose. Both create a cache.db with a metadata table and insert a jsonl_hash key. The cli test file also duplicates the SHA256 hashing logic from computeTestHash (doctor package, lines 68-71) inline in setupDoctorProject, setupDoctorProjectWithContent, and setupDoctorProjectWithContentStale.
  RECOMMENDATION: Create a shared internal/doctor/testutil_test.go or internal/testutil package exporting CreateTestCache and ComputeTestHash, then have both doctor and cli tests import it. Alternatively, since Go test helpers cannot cross package boundaries without export, consider an internal/testutil package with exported helpers. This would eliminate 5 near-duplicate setup functions in the cli test file (setupDoctorProject, setupDoctorProjectStale, setupDoctorProjectWithContent, setupDoctorProjectWithContentStale) that re-implement doctor test patterns.
SUMMARY: One medium-severity cross-package test helper duplication found between doctor and cli test files for cache setup and hash computation. Production code is clean after three prior consolidation cycles.
