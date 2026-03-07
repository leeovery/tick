AGENT: duplication
FINDINGS:
- FINDING: Repeated Getwd + error handling block across 13 handler methods
  SEVERITY: low
  FILES: internal/cli/app.go:131-133, internal/cli/app.go:140-142, internal/cli/app.go:149-151, internal/cli/app.go:162-164, internal/cli/app.go:170-172, internal/cli/app.go:180-182, internal/cli/app.go:193-195, internal/cli/app.go:206-208, internal/cli/app.go:215-217, internal/cli/app.go:228-230, internal/cli/app.go:276-278, internal/cli/dep.go:14-16, internal/cli/note.go:15-17
  DESCRIPTION: The identical 3-line block `dir, err := a.Getwd(); if err != nil { return fmt.Errorf("could not determine working directory: %w", err) }` is copy-pasted 13 times across every handler method. While each instance is small, the aggregate is ~39 lines of pure duplication.
  RECOMMENDATION: Extract a private method `func (a *App) resolveDir() (string, error)` that wraps Getwd with the error message. Each handler becomes a single-line call. Low severity because each instance is only 3 lines and this is a standard Go DI pattern, but the sheer repetition count makes it a clean extraction candidate.

- FINDING: Structurally identical ValidateTags and ValidateRefs functions
  SEVERITY: low
  FILES: internal/task/tags.go:43-61, internal/task/refs.go:43-61
  DESCRIPTION: ValidateTags and ValidateRefs follow an identical structure: check empty, deduplicate, validate each item in a loop, check count against a max constant. The only differences are the entity name in error strings, the deduplicate function called, the per-item validator, and the max constant. Each is ~18 lines.
  RECOMMENDATION: Could extract a generic `validateSlice(items []string, dedup func, validate func(string) error, max int, name string) error` helper in helpers.go. However, with only 2 instances and the shared `deduplicateStrings` foundation already in place, this is borderline -- the existing code is clear and direct. Low severity.

SUMMARY: No high or medium severity duplication remains. The cycle-2 fixes (queryStringColumn/queryRelatedTasks helpers, ParseRefs delegation) addressed the significant patterns. Two low-severity patterns exist: the Getwd boilerplate repeated 13 times across handlers, and the structurally parallel ValidateTags/ValidateRefs functions. Both are clean extraction candidates but individually small.
