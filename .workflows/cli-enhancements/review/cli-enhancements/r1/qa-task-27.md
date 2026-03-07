TASK: cli-enhancements-5-3 -- Extract shared buildStringListSection in toon_formatter.go

ACCEPTANCE CRITERIA:
- buildTagsSection and buildRefsSection delegate to a single shared function
- TOON formatter output is identical before and after (no behavioral change)

STATUS: Complete

SPEC CONTEXT: Tags and refs are both []string fields displayed in show output. The toon formatter renders them as TOON array sections with "name[count]:" header and indented values. Both followed an identical pattern prior to extraction.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:203-212` -- `buildStringListSection(name string, items []string) string` shared helper
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:193-196` -- `buildTagsSection` delegates to `buildStringListSection("tags", tags)`
  - `/Users/leeovery/Code/tick/internal/cli/toon_formatter.go:198-201` -- `buildRefsSection` delegates to `buildStringListSection("refs", refs)`
- Notes: Implementation matches the analysis task specification exactly. The shared helper uses strings.Builder, writes "name[count]:" header, and iterates with "\n  " prefix per item. Both wrapper functions are preserved as thin one-line delegators with clear godoc comments, maintaining the call sites in FormatTaskDetail unchanged (lines 93 and 98).

TESTS:
- Status: Adequate
- Coverage: Existing toon formatter tests cover both tags and refs rendering through the new shared helper:
  - Line 480: "it displays tags in toon format show output" -- verifies "tags[2]:" header and indented values
  - Line 509: "it omits tags section in toon format when task has no tags" -- verifies empty omission
  - Line 530: "it displays refs in toon show output" -- verifies "refs[2]:" header and indented values
  - Line 559: "it omits refs section in toon format when task has no refs" -- verifies empty omission
- Notes: This is a pure refactoring task with no behavioral change. The analysis task explicitly states "All existing toon formatter tests pass unchanged." The existing tests exercise both code paths through the shared helper and are sufficient. No new tests needed or expected.

CODE QUALITY:
- Project conventions: Followed. Unexported helper function, godoc comments on all functions, consistent naming conventions with other build* helpers in the file.
- SOLID principles: Good. Single responsibility -- one function for one concern (rendering string lists as TOON sections). Open/closed -- adding a new string-list field requires only a one-line wrapper calling the shared helper.
- Complexity: Low. The shared helper is 5 lines with a single loop. The wrapper functions are each one line.
- Modern idioms: Yes. Uses strings.Builder for efficient string building, fmt.Fprintf for formatted writes.
- Readability: Good. Function name clearly conveys purpose. Wrapper functions make the call site self-documenting.
- Issues: None.

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The wrapper functions (buildTagsSection, buildRefsSection) could theoretically be inlined at the call sites in FormatTaskDetail, calling buildStringListSection directly. However, keeping them provides semantic clarity and matches the pattern of other build* helpers in the file, so this is a reasonable design choice.
