AGENT: duplication
FINDINGS:
- FINDING: DeduplicateTags and DeduplicateRefs are structurally identical
  SEVERITY: medium
  FILES: internal/task/tags.go:38, internal/task/refs.go:38
  DESCRIPTION: DeduplicateTags and DeduplicateRefs implement the same algorithm -- iterate a slice, normalize each element, skip empties, deduplicate by seen-map, preserve first-occurrence order. The only differences are: (1) NormalizeTag vs strings.TrimSpace as the normalize step, and (2) the field name in the map key. Both functions are ~14 lines of identical structure. ValidateTags (line 55) and ValidateRefs (line 55) follow the same pattern as well: early-return on empty, call Deduplicate*, validate each element, check max count. That adds another ~18 lines of near-duplicate logic.
  RECOMMENDATION: Extract a generic deduplicateSlice helper (or use a shared function with a normalizer callback) in internal/task/. For example: func deduplicateStrings(items []string, normalize func(string) string) []string. Both DeduplicateTags and DeduplicateRefs become one-line wrappers. Similarly, a validateCollection helper could take the deduplicate function, per-item validator, and max count, reducing ValidateTags/ValidateRefs to thin wrappers.

- FINDING: buildTagsSection and buildRefsSection are identical except for the section name
  SEVERITY: medium
  FILES: internal/cli/toon_formatter.go:194, internal/cli/toon_formatter.go:206
  DESCRIPTION: Both functions build a TOON string-list section with the pattern: write "name[count]:", then iterate and write "\n  " + value. They differ only in the section name ("tags" vs "refs") and the parameter name. Each is 8 lines of identical logic.
  RECOMMENDATION: Extract a shared buildStringListSection(name string, items []string) string function in toon_formatter.go. Both buildTagsSection and buildRefsSection become calls to it, or are removed entirely.

- FINDING: Tags/Refs/Type validation blocks duplicated between create.go and update.go
  SEVERITY: medium
  FILES: internal/cli/create.go:130-160, internal/cli/update.go:159-202
  DESCRIPTION: Both RunCreate and RunUpdate contain nearly identical validation blocks for --type (normalize, ValidateTypeNotEmpty, ValidateType), --tags (DeduplicateTags, check empty, ValidateTags), and --refs (DeduplicateRefs, check empty, ValidateRefs). The create and update flows each independently implement the same sequence of normalize-then-validate-then-check-empty for each of these three fields. The pattern is repeated 3 times across 2 files = 6 blocks of structurally identical validation (~5-8 lines each).
  RECOMMENDATION: Extract field-specific validation helpers into helpers.go. For example: validateTypeFlag(value string) (string, error), validateTagsFlag(tags []string) ([]string, error), validateRefsFlag(refs []string) ([]string, error). Each encapsulates the normalize+empty-check+validate sequence. Both create.go and update.go call these one-liners.

- FINDING: Test runner helper functions are copy-pasted across test files
  SEVERITY: low
  FILES: internal/cli/create_test.go:60, internal/cli/update_test.go:14, internal/cli/note_test.go:14
  DESCRIPTION: runCreate, runUpdate, and runNote are structurally identical: create bytes.Buffer pair, construct App with IsTTY=true, build fullArgs with command prefix, call app.Run, return stdout/stderr/exitCode. The only difference is the command name injected into the args slice ("create", "update", "note"). Each is ~12 lines.
  RECOMMENDATION: Extract a shared runCommand(t, dir, command string, args ...string) helper into the existing test helpers file (create_test.go already has setupTickProject). The three functions become one-line wrappers or are replaced entirely.

SUMMARY: Four duplication patterns found. The most impactful are the parallel DeduplicateTags/DeduplicateRefs functions in internal/task/ and the repeated type/tags/refs validation blocks across create.go and update.go, both of which risk drift as these features evolve. The TOON section builders and test helpers are lower risk but still worth consolidating.
