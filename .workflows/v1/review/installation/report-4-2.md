TASK: Extract loadScript helper in install_test.go

ACCEPTANCE CRITERIA:
- A single loadScript helper exists in install_test.go
- No inline os.ReadFile calls for install.sh remain in install_test.go
- All existing tests pass unchanged

STATUS: Complete

SPEC CONTEXT: This is a cycle 2 analysis/refactoring task. The install_test.go file had 8 occurrences of a 4-line read-file-to-string pattern (scriptPath, ReadFile, error check, string conversion). The task extracts these into a single loadScript helper to reduce duplication and simplify future test additions.

IMPLEMENTATION:
- Status: Implemented
- Location: /Users/leeovery/Code/tick/scripts/install_test.go:21-28
- Notes: The loadScript function matches the planned implementation exactly. It uses t.Helper(), calls os.ReadFile(scriptPath(t)), fatals with a clear message on error, and returns string(data). All 8 former inline occurrences now call loadScript(t) at lines 81, 89, 412, 822, 831, 840, 855, and 1025.

TESTS:
- Status: Adequate
- Coverage: This is a test-only refactoring task. The existing tests exercise loadScript implicitly -- 8 tests call it and would fail if it were broken (e.g., wrong path, missing file, incorrect return). No separate unit test for loadScript is needed; the helper is validated through its callers.
- Notes: The remaining os.ReadFile calls in the file (line 573 reads an installed binary, line 651 reads a brew log file) are unrelated to install.sh content and are correctly left as-is.

CODE QUALITY:
- Project conventions: Followed. Uses t.Helper() per CLAUDE.md patterns. Follows the same pattern as loadFormula and loadWorkflow helpers mentioned in the task description.
- SOLID principles: Good. Single responsibility -- one function, one job (load script content for tests).
- Complexity: Low. Linear 5-line function with a single error path.
- Modern idioms: Yes. Uses os.ReadFile (not ioutil.ReadFile), t.Helper(), t.Fatalf.
- Readability: Good. Function name is descriptive, doc comment present on line 20, error message is clear.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- None
