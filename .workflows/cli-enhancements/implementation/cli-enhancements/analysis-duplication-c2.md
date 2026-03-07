AGENT: duplication
FINDINGS:
- FINDING: Near-identical row-scanning blocks in queryShowData (tags and refs)
  SEVERITY: medium
  FILES: internal/cli/show.go:150-168, internal/cli/show.go:171-189
  DESCRIPTION: The tags query block (lines 150-168) and refs query block (lines 171-189) in queryShowData are structurally identical -- both query a junction table for a single string column by task_id, scan rows in a loop, append to a string slice, and check rows.Err(). The only differences are table name (task_tags vs task_refs), column name (tag vs ref), variable names, and error message prefix. This is ~19 lines duplicated verbatim with only identifier substitution.
  RECOMMENDATION: Extract a helper function like `queryStringColumn(db *sql.DB, query string, id string, errPrefix string) ([]string, error)` in show.go. Both call sites become single-line calls. Similarly, the blocked_by block (lines 108-126) and children block (lines 129-147) both scan into RelatedTask with identical structure -- a `queryRelatedTasks(db *sql.DB, query string, id string, errPrefix string) ([]RelatedTask, error)` helper would consolidate those as well. Total: ~76 lines of query+scan boilerplate reduced to ~30 lines (2 helpers + 4 call sites).

- FINDING: ParseRefs reimplements ValidateRefs logic instead of calling it
  SEVERITY: low
  FILES: internal/task/refs.go:43-61, internal/task/refs.go:65-84
  DESCRIPTION: ParseRefs (lines 65-84) duplicates the validate-each-ref and check-count logic from ValidateRefs (lines 43-61). After splitting on comma and deduplicating, ParseRefs independently iterates refs to call ValidateRef and checks the count against maxRefsPerTask -- exactly what ValidateRefs already does. The error format strings for the count check are identical between the two functions.
  RECOMMENDATION: Have ParseRefs call ValidateRefs on the deduped slice instead of reimplementing the validation loop and count check. ValidateRefs will re-dedup (harmless no-op on already-deduped input) and validate. Reduces ParseRefs from 19 lines to ~10 lines and ensures validation logic stays in one place.

SUMMARY: Two duplication patterns found. The most impactful is in queryShowData where 4 query-scan blocks (~76 lines) follow identical patterns and could be consolidated into 2 helper functions. A minor case exists in refs.go where ParseRefs reimplements ValidateRefs logic instead of calling it.
