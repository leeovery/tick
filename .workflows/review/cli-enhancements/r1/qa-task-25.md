TASK: cli-enhancements-5-1 -- Add type column to queryShowData SQL query

ACCEPTANCE CRITERIA:
- showData struct includes a type field
- SQL SELECT in queryShowData includes type column
- showDataToTaskDetail populates Task.Type from showData
- tick show output displays the correct type for tasks that have one
- Post-mutation output (create, update, note) displays the correct type

STATUS: Complete

SPEC CONTEXT: The specification requires that show output displays the type field, and list output includes a Type column. The analysis cycle 1 identified that queryShowData originally omitted the type column from its SQL query, causing all detail output (show, create, update, note) to render type as empty regardless of actual value. This was classified as high severity.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/show.go:19` -- showData struct has `taskType string` field
  - `/Users/leeovery/Code/tick/internal/cli/show.go:72` -- typePtr declared alongside descPtr/parentPtr/closedPtr
  - `/Users/leeovery/Code/tick/internal/cli/show.go:74` -- SQL SELECT includes `type` column: `SELECT id, title, status, priority, type, description, parent, created, updated, closed FROM tasks WHERE id = ?`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:76` -- typePtr scanned in correct positional order (5th column)
  - `/Users/leeovery/Code/tick/internal/cli/show.go:84-86` -- nil check and assignment: `data.taskType = *typePtr`
  - `/Users/leeovery/Code/tick/internal/cli/show.go:219` -- showDataToTaskDetail maps `d.taskType` to `t.Type`
- Notes: All six acceptance criteria are met. The implementation follows the exact same nullable-pointer pattern used for description, parent, and closed fields. The analysis cycle 2 architecture report confirmed this fix as correct.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/internal/cli/list_show_test.go:677-693` -- "it displays type in show output when task has a type": creates task with Type "bug", verifies show output contains "Type:     bug"
  - `/Users/leeovery/Code/tick/internal/cli/list_show_test.go:695-711` -- "it displays dash for type in show output when task has no type": creates task without type, verifies show output contains "Type:     -"
  - `/Users/leeovery/Code/tick/internal/cli/list_show_test.go:713-724` -- "it displays type in post-mutation output after create with --type": creates via CLI with --type feature, verifies output contains "Type:     feature"
  - `/Users/leeovery/Code/tick/internal/cli/list_show_test.go:726-742` -- "it displays type in post-mutation output after update": task with Type "feature", runs update, verifies output preserves "Type:     feature"
- Notes: Tests cover all acceptance criteria scenarios: type present, type absent (dash), post-create mutation output, and post-update mutation output. Tests would fail if the type column were removed from the SQL query. Not over-tested -- each test verifies a distinct path through the show/mutation output.

CODE QUALITY:
- Project conventions: Followed -- uses the established nullable-pointer scan pattern (typePtr alongside descPtr/parentPtr/closedPtr), stdlib testing with t.Run subtests, error wrapping with fmt.Errorf
- SOLID principles: Good -- queryShowData remains a single-purpose function; showDataToTaskDetail handles conversion separately
- Complexity: Low -- the change adds one field to a struct, one column to a SQL query, one scan variable, one nil check, and one field mapping
- Modern idioms: Yes -- idiomatic Go with pointer-based nullable handling for SQL
- Readability: Good -- field ordering in showData matches SQL column order; typePtr follows the same pattern as descPtr/parentPtr/closedPtr making it easy to understand
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
