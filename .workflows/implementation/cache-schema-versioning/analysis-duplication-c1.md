AGENT: duplication
FINDINGS:
- FINDING: Repeated cache-tampering test setup pattern across store_test.go
  SEVERITY: low
  FILES: internal/storage/store_test.go:1284-1294, internal/storage/store_test.go:1357-1366, internal/storage/store_test.go:1468-1477, internal/storage/store_test.go:1544-1582, internal/storage/store_test.go:1652-1661
  DESCRIPTION: Five test cases in TestStoreSchemaVersionCheck and TestStoreSchemaVersionUpgrade follow the same pattern: prime the cache via store.Query, close the store, open cache.db with raw sql.Open, tamper with metadata (update/delete schema_version or corrupt the table), close the raw db, then reopen via NewStore and verify recovery. The prime-close-tamper-reopen sequence is repeated with minor variations (different SQL statements for the tampering step). Each instance is 15-25 lines of setup boilerplate.
  RECOMMENDATION: This is borderline -- the tamper step differs meaningfully in each case (UPDATE to wrong value, DELETE row, DROP+CREATE table, DROP+CREATE tasks table). Extracting a helper like tamperCacheMetadata(t, cachePath, sql string) would reduce the boilerplate to a single call per test but the diversity of tampering makes a single helper awkward. No action recommended at this severity -- flagging for awareness only.
- FINDING: Duplicate computeHash implementations in production and test code
  SEVERITY: low
  FILES: internal/storage/cache.go:280-283, internal/storage/cache_test.go:1516-1519
  DESCRIPTION: cache.go has computeHash(data []byte) string and cache_test.go has computeTestHash(data []byte) string -- identical SHA256 implementations. The test version exists because computeHash is unexported.
  RECOMMENDATION: No action needed. This is intentional Go convention -- the test helper independently computes the expected hash to validate the production function rather than depending on it. Coupling the test to the production function would defeat the purpose. This is the correct pattern.
SUMMARY: No significant duplication requiring consolidation. The production code (cache.go, store.go) has clean separation with no repeated logic. The two metadata INSERT OR REPLACE calls in Rebuild() store different keys and are not candidates for extraction. Test files show some repeated setup patterns but these are within normal bounds for Go test code and the variations are meaningful.
