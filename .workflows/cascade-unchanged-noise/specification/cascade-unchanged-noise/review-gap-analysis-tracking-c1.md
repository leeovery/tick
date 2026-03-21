---
status: in-progress
created: 2026-03-21
cycle: 1
phase: Gap Analysis
topic: cascade-unchanged-noise
---

# Review Tracking: cascade-unchanged-noise - Gap Analysis

## Findings

### 1. Testing section incomplete -- missing affected tests across multiple files

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Testing

**Details**:
The Testing section names only 2 subtests to remove (both in `TestBuildCascadeResult` in `cascade_formatter_test.go`). However, `Unchanged`/`UnchangedEntry` references appear in tests across 4 files that will need modification:

- `cascade_formatter_test.go`: Beyond the 2 named subtests, at least 8 additional test cases construct `CascadeResult` with `Unchanged` fields or assert on unchanged rendering (e.g., "it renders mixed cascaded and unchanged children", "it renders unchanged terminal grandchildren in tree", JSON unchanged array assertions, "all formatters handle both empty cascaded and unchanged").
- `transition_test.go` (line 514): "it includes unchanged terminal children in cascade output" -- an integration test that sets up tasks and asserts `stdout` contains "unchanged". This test must be removed or rewritten as the negative-case test the spec calls for.
- `format_test.go` (lines 387, 440, 462-466): `TestCascadeResultStruct` and `FormatCascadeTransition` tests construct `CascadeResult` with `Unchanged` fields and assert on them.
- `helpers_test.go`: Uses `buildCascadeResult` but does not assert on `Unchanged` directly, so these should be fine after the type changes.

An implementer following only the Testing section as written would miss breaking tests and get compile errors from the type removal.

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 2. Dead code not called out -- involvedIDs map becomes unused after removal

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Fix, item 1

**Details**:
The Fix section says to remove the unchanged collection loop at lines 117-135. However, lines 110-115 build an `involvedIDs` map that exists solely to support that loop. After removing lines 117-135, `involvedIDs` becomes dead code and will cause a Go compile error (`declared and not used`). The spec should include lines 110-115 in the removal scope (i.e., lines 110-135, not 117-135).

**Proposed Addition**:

**Resolution**: Pending
**Notes**:

---

### 3. Fix file list omits test files that will fail to compile

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Fix

**Details**:
The Fix section lists 5 source files to modify. Removing `UnchangedEntry` type and `Unchanged` field from `CascadeResult` (item 2) will cause compile errors in every test file that references these types. The Fix section should either list the test files as additional items or note that test files referencing the removed types must also be updated. Without this, the file list gives a false impression of the change's blast radius.

Affected test files: `cascade_formatter_test.go`, `format_test.go`, `transition_test.go` (and trivially `helpers_test.go` if any implicit usage).

**Proposed Addition**:

**Resolution**: Pending
**Notes**: This is closely related to Finding 1 but concerns the Fix section rather than Testing. The Fix section's numbered file list reads as exhaustive but isn't.
