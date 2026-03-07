AGENT: duplication
FINDINGS:
- FINDING: Find-task-by-index pattern duplicated within dep.go
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/dep.go:73-81, /Users/leeovery/Code/tick/internal/cli/dep.go:141-148
  DESCRIPTION: RunDepAdd and RunDepRm contain identical 8-line blocks that find a task by ID in the tasks slice (taskIdx := -1, for-range with ID comparison, break, check -1 and return "not found" error). This was reported in C3 and remains unfixed. The two instances are byte-for-byte identical within the same file.
  RECOMMENDATION: Extract a findTaskIndex(tasks []task.Task, id string) (int, error) helper in helpers.go. Both RunDepAdd and RunDepRm call it. Low priority since it is only ~8 lines duplicated within a single file.

- FINDING: Getwd + error-wrap boilerplate repeated across all app.go handler methods
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/app.go:94-97, /Users/leeovery/Code/tick/internal/cli/app.go:103-106, /Users/leeovery/Code/tick/internal/cli/app.go:111-114, /Users/leeovery/Code/tick/internal/cli/app.go:125-128, /Users/leeovery/Code/tick/internal/cli/app.go:134-137, /Users/leeovery/Code/tick/internal/cli/app.go:143-146, /Users/leeovery/Code/tick/internal/cli/app.go:156-159, /Users/leeovery/Code/tick/internal/cli/app.go:169-172, /Users/leeovery/Code/tick/internal/cli/app.go:178-181, /Users/leeovery/Code/tick/internal/cli/app.go:187-190, /Users/leeovery/Code/tick/internal/cli/dep.go:14-17
  DESCRIPTION: All 11 handler methods in app.go (plus handleDep in dep.go) repeat the same 4-line Getwd + error-wrap block: dir, err := a.Getwd(); if err != nil { return fmt.Errorf("could not determine working directory: %w", err) }. This totals ~44 lines of identical code. Each handler is trivially thin (5-7 lines total), so the duplication is proportionally significant per handler but the individual blocks are small.
  RECOMMENDATION: Extract a private method like (a *App) workDir() (string, error) that wraps a.Getwd() with the standard error message. Each handler saves 3 lines. Low priority since the pattern is idiomatic Go and each handler remains readable as-is.

SUMMARY: The C3 medium-severity finding (post-mutation output pattern) was successfully consolidated into outputMutationResult in helpers.go. Two low-severity findings remain: the find-task-by-index duplication within dep.go (carried from C3), and a new finding of repeated Getwd boilerplate across 11 handler methods in app.go. No high-severity duplication detected.
