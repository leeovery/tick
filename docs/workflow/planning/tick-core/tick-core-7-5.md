---
id: tick-core-7-5
phase: 7
status: completed
created: 2026-02-10
---

# Consolidate duplicate relatedTask struct into RelatedTask

**Problem**: show.go defines an unexported `relatedTask{id, title, status string}` struct (lines 30-34) used in `queryShowData`, while format.go defines the exported `RelatedTask{ID, Title, Status string}` (lines 88-92) with identical fields. The `showDataToTaskDetail` function (show.go lines 176-191) loops through each `relatedTask` and converts it to a `RelatedTask` by copying field by field. The two structs are structurally identical -- the only difference is export visibility and field naming.

**Solution**: Use `RelatedTask` directly in `queryShowData` instead of the unexported `relatedTask`. SQL Scan calls can target `RelatedTask` fields directly (`&r.ID`, `&r.Title`, `&r.Status`). This eliminates the `relatedTask` struct and the two conversion loops in `showDataToTaskDetail`.

**Outcome**: One struct for related task data. Approximately 15 lines of mapping code removed. No intermediate type conversion needed between query and formatting layers.

**Do**:
1. In `internal/cli/show.go`, change the `showData` struct to use `[]RelatedTask` instead of `[]relatedTask` for the `blockedBy` and `children` fields
2. Update `queryShowData` to scan directly into `RelatedTask` fields (`&r.ID`, `&r.Title`, `&r.Status`)
3. Remove the unexported `relatedTask` struct definition from show.go
4. Simplify `showDataToTaskDetail` to directly assign the `[]RelatedTask` slices instead of converting field by field
5. Run all tests to verify no behavioral changes

**Acceptance Criteria**:
- The unexported `relatedTask` struct no longer exists in show.go
- `queryShowData` populates `RelatedTask` directly
- `showDataToTaskDetail` no longer has field-by-field conversion loops
- All existing show and format tests pass unchanged

**Tests**:
- Test that tick show output is unchanged after refactor (covered by existing show tests)
- Test that queryShowData correctly populates RelatedTask fields
