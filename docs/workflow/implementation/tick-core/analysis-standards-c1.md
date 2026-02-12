AGENT: standards
FINDINGS:
- FINDING: Missing dependency validation on create and update --blocked-by/--blocks
  SEVERITY: high
  FILES: /Users/leeovery/Code/tick/internal/cli/create.go:146, /Users/leeovery/Code/tick/internal/cli/update.go:195
  DESCRIPTION: The spec states (line 403): "Validate when adding/modifying blocked_by - reject invalid dependencies at write time, before persisting to JSONL." The `tick dep add` command correctly calls `task.ValidateDependency()` for cycle detection and child-blocked-by-parent checks. However, `tick create --blocked-by` only calls `validateRefs()` which checks existence and self-reference but NOT cycles or child-blocked-by-parent. Similarly, `tick create --blocks` and `tick update --blocks` append to target tasks' blocked_by arrays without any dependency validation. This means invalid dependency graphs (cycles, child-blocked-by-parent) can be persisted through create and update commands, violating the spec's write-time validation rule.
  RECOMMENDATION: Call `task.ValidateDependency()` or `task.ValidateDependencies()` in `RunCreate` for both `--blocked-by` and `--blocks` targets after building the new task, and in `RunUpdate` for `--blocks` targets. The full task list (including the new/modified task) must be passed to enable proper graph analysis.

- FINDING: Transition output uses ASCII arrow instead of spec's Unicode arrow
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:91, /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:123, /Users/leeovery/Code/tick/internal/cli/json_formatter.go:111
  DESCRIPTION: The spec (line 639) specifies transition output as `tick-a3f2b7: open -> in_progress` using the Unicode right arrow U+2192. The implementation uses ASCII `->`. Notably, the cycle detection error messages in dependency.go correctly use `\u2192` matching the spec (line 410), making the codebase internally inconsistent. The transition formatters all use the ASCII form.
  RECOMMENDATION: Change the `->` in FormatTransition methods to use the Unicode arrow to match the spec and be consistent with dependency.go's cycle error messages. Alternatively, if ASCII is preferred for agent consumption, update the spec.

- FINDING: tick doctor command not implemented
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/app.go:57
  DESCRIPTION: The spec command reference (line 455) lists `tick doctor` as "Run diagnostics and validation." The spec also references it in edge cases (lines 427-430) for detecting orphaned children and parents done before children. No implementation exists; the command is not routed in app.go's switch statement. The command is not in the provided implementation file list either, suggesting it may be intentionally deferred.
  RECOMMENDATION: If intentionally deferred, document that `tick doctor` is out of scope for this cycle. If it should be part of this implementation, add the handler. The spec treats it as part of the core command set.

- FINDING: Child-blocked-by-parent error message missing explanatory second line
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/task/dependency.go:36
  DESCRIPTION: The spec (lines 407-408) defines the child-blocked-by-parent error as a two-line message including the rationale: "Cannot add dependency - tick-child cannot be blocked by its parent tick-epic" followed by "(would create unworkable task due to leaf-only ready rule)". The implementation only outputs the first line (and uses lowercase "cannot" vs spec's "Cannot").
  RECOMMENDATION: Add the second line explaining why the constraint exists, matching the spec format: `(would create unworkable task due to leaf-only ready rule)`.

SUMMARY: The most significant drift is the missing dependency validation (cycle detection and child-blocked-by-parent) in `tick create --blocked-by/--blocks` and `tick update --blocks`, which allows invalid dependency graphs to be persisted despite the spec requiring write-time validation for all blocked_by modifications. The `tick doctor` command is also unimplemented. Two low-severity formatting differences exist in transition arrow characters and error message wording.
