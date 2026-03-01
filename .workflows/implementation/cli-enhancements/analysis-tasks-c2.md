---
topic: cli-enhancements
cycle: 2
total_proposed: 3
---
# Analysis Tasks: cli-enhancements (Cycle 2)

## Task 1: Resolve partial IDs for list --parent
status: approved
severity: high
sources: architecture

**Problem**: `list --parent` normalizes the ID via `task.NormalizeID()` but never resolves it through `store.ResolveID()`. The raw normalized value is passed directly to SQL (`WHERE id = ?`), so `tick list --parent a3f` fails to find `tick-a3f1b2`. Every other ID-accepting parameter (`create --parent`, `update --parent`, `dep add`, positional args) correctly resolves partial IDs. The specification states partial ID resolution "applies everywhere an ID is accepted: positional args, --parent, --blocked-by, --blocks".

**Solution**: In `RunList`, resolve `filter.Parent` through `store.ResolveID()` before entering the `store.Query()` closure, following the same pattern used in `RunUpdate` and `RunCreate`.

**Outcome**: `tick list --parent a3f` correctly resolves to `tick-a3f1b2` (or errors on ambiguity/not-found), consistent with all other commands.

**Do**:
1. In `internal/cli/list.go`, in the `RunList` function, after `store` is opened (line ~150) and before the `store.Query()` call (line ~166), add partial ID resolution for `filter.Parent`:
   ```go
   if filter.Parent != "" {
       filter.Parent, err = store.ResolveID(filter.Parent)
       if err != nil {
           return err
       }
   }
   ```
2. Remove the parent-existence check inside the `store.Query()` closure (lines 170-176) since `ResolveID` already validates that the task exists.

**Acceptance Criteria**:
- `tick list --parent <partial-id>` resolves the partial ID to a full ID before querying
- Ambiguous partial IDs produce an error listing matching IDs
- Non-existent IDs produce a "not found" error
- Full IDs continue to work as before

**Tests**:
- Test that `list --parent` with a unique prefix resolves correctly and returns children
- Test that `list --parent` with an ambiguous prefix returns an appropriate error
- Test that `list --parent` with a non-matching prefix returns a not-found error

## Task 2: Extract query-scan helpers in show.go
status: approved
severity: medium
sources: duplication

**Problem**: `queryShowData` in `internal/cli/show.go` contains four structurally identical query-scan blocks: blocked_by (lines 108-126), children (lines 129-147), tags (lines 150-168), and refs (lines 171-189). Each block runs a query, iterates rows, scans into a variable, appends to a slice, and checks `rows.Err()`. The only differences are the SQL query, scan target type, variable names, and error message prefix. This is approximately 76 lines of boilerplate.

**Solution**: Extract two helper functions: (1) `queryStringColumn(db *sql.DB, query string, id string) ([]string, error)` for tags and refs, and (2) `queryRelatedTasks(db *sql.DB, query string, id string) ([]RelatedTask, error)` for blocked_by and children. Replace the four inline blocks with four single-line calls.

**Outcome**: The four query-scan blocks (~76 lines) reduce to two helper functions (~30 lines) plus four call sites, improving readability and reducing maintenance surface.

**Do**:
1. In `internal/cli/show.go`, add a helper function:
   ```go
   func queryStringColumn(db *sql.DB, query string, id string) ([]string, error) {
       rows, err := db.Query(query, id)
       if err != nil {
           return nil, err
       }
       defer rows.Close()
       var result []string
       for rows.Next() {
           var val string
           if err := rows.Scan(&val); err != nil {
               return nil, err
           }
           result = append(result, val)
       }
       return result, rows.Err()
   }
   ```
2. Add a second helper function:
   ```go
   func queryRelatedTasks(db *sql.DB, query string, id string) ([]RelatedTask, error) {
       rows, err := db.Query(query, id)
       if err != nil {
           return nil, err
       }
       defer rows.Close()
       var result []RelatedTask
       for rows.Next() {
           var r RelatedTask
           if err := rows.Scan(&r.ID, &r.Title, &r.Status); err != nil {
               return nil, err
           }
           result = append(result, r)
       }
       return result, rows.Err()
   }
   ```
3. Replace the four inline blocks in `queryShowData` with calls to these helpers:
   ```go
   data.blockedBy, err = queryRelatedTasks(db, `SELECT t.id, t.title, t.status FROM dependencies d JOIN tasks t ON d.blocked_by = t.id WHERE d.task_id = ? ORDER BY t.id`, id)
   if err != nil { return fmt.Errorf("failed to query dependencies: %w", err) }

   data.children, err = queryRelatedTasks(db, `SELECT id, title, status FROM tasks WHERE parent = ? ORDER BY id`, id)
   if err != nil { return fmt.Errorf("failed to query children: %w", err) }

   data.tags, err = queryStringColumn(db, `SELECT tag FROM task_tags WHERE task_id = ? ORDER BY tag`, id)
   if err != nil { return fmt.Errorf("failed to query tags: %w", err) }

   data.refs, err = queryStringColumn(db, `SELECT ref FROM task_refs WHERE task_id = ? ORDER BY ref`, id)
   if err != nil { return fmt.Errorf("failed to query refs: %w", err) }
   ```

**Acceptance Criteria**:
- All four query-scan blocks in `queryShowData` use the extracted helpers
- `tick show` output is unchanged for all cases (deps, children, tags, refs)
- No new exported symbols introduced (helpers are unexported)

**Tests**:
- Existing `show` tests pass without modification (no behavioral change)

## Task 3: ParseRefs should delegate to ValidateRefs
status: approved
severity: low
sources: duplication

**Problem**: `ParseRefs` in `internal/task/refs.go` (lines 65-84) reimplements the validation loop and count check that `ValidateRefs` (lines 43-61) already provides. After splitting and deduplicating, `ParseRefs` independently iterates refs to call `ValidateRef` and checks the count against `maxRefsPerTask` -- exactly what `ValidateRefs` does. The error format strings are identical.

**Solution**: Have `ParseRefs` call `ValidateRefs` on the deduplicated slice instead of reimplementing the validation logic inline.

**Outcome**: `ParseRefs` drops from ~19 lines to ~10, and validation logic for refs lives in one place (`ValidateRefs`).

**Do**:
1. In `internal/task/refs.go`, replace the validation loop and count check in `ParseRefs` with a single call to `ValidateRefs`:
   ```go
   func ParseRefs(input string) ([]string, error) {
       if strings.TrimSpace(input) == "" {
           return nil, errors.New("refs input cannot be empty")
       }
       parts := strings.Split(input, ",")
       refs := DeduplicateRefs(parts)
       if err := ValidateRefs(refs); err != nil {
           return nil, err
       }
       return refs, nil
   }
   ```

**Acceptance Criteria**:
- `ParseRefs` delegates validation to `ValidateRefs` instead of reimplementing it
- All existing `ParseRefs` tests pass without modification

**Tests**:
- Existing refs tests pass (no behavioral change)
