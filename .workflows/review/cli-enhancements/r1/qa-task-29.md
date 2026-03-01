TASK: cli-enhancements-5-5 -- Consolidate ResolveID into a single Query call

ACCEPTANCE CRITERIA:
- ResolveID makes exactly one s.Query() call regardless of whether it does exact or prefix matching
- All existing ResolveID tests pass unchanged
- Ambiguity and not-found error behavior is preserved

STATUS: Complete

SPEC CONTEXT: ResolveID resolves user-supplied partial or full task IDs to canonical full IDs. The spec requires exact full-ID match priority, prefix matching with minimum 3 hex chars, ambiguity detection, and case-insensitive input normalization. The analysis finding (architecture, medium severity) identified that the original implementation called s.Query() up to twice -- once for exact match and once for prefix search -- doubling lock acquisition and cache-freshness I/O on every command invocation.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/internal/storage/store.go:288-353
- Notes: The method uses exactly one s.Query() call (line 305). Inside the callback, exact match is attempted first when hex length is 6 (lines 309-317), then prefix search runs as a fallback (lines 319-346). The comment on line 303 explicitly documents the consolidation: "Single query call: exact match first (6 hex chars), then prefix search fallback." All original behavior (ambiguity errors, not-found errors, prefix stripping, case normalization) is preserved within the single callback. No second s.Query() call exists anywhere in the function.

TESTS:
- Status: Adequate
- Coverage: /Users/leeovery/Code/tick/internal/storage/resolve_id_test.go contains 12 subtests covering: unique 3-char prefix, tick- prefix stripping, case normalization, case-insensitive prefix stripping, exact full-ID bypass without ambiguity, short prefix errors (2 chars, 1 char), ambiguous prefix with matching ID listing, not-found for non-matching prefix, 4-char and 5-char unique prefix resolution, 6-char fallback to prefix search on miss, empty string, tick- with fewer than 3 hex chars, and original input preservation in error messages.
- Notes: Tests are behavioral and do not test implementation details (e.g., they do not assert how many Query calls are made). This is appropriate -- the consolidation is a performance refactoring, and the tests verify that behavior is unchanged. Each subtest covers a distinct scenario without redundancy.

CODE QUALITY:
- Project conventions: Followed. Uses project's error wrapping pattern, Query callback pattern, and comment style.
- SOLID principles: Good. Single responsibility maintained. The function does one thing: resolve input to a full task ID.
- Complexity: Low. Linear flow with one conditional branch (exact match for 6-char hex) and a switch statement for match count.
- Modern idioms: Yes. Proper use of defer rows.Close(), sql.QueryRow for single-row queries, sql.Query for multi-row, and rows.Err() check.
- Readability: Good. Well-commented, clear variable names (hex, fullID, resolved, matches), logical code structure.
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
