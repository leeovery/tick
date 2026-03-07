AGENT: standards
FINDINGS:
- FINDING: --blocks on update allows duplicate blocked_by entries
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/helpers.go:37-46, /Users/leeovery/Code/tick/internal/cli/update.go:184-185
  DESCRIPTION: The `applyBlocks` helper blindly appends `sourceID` to target tasks' `BlockedBy` slices without checking for duplicates. When `tick update T1 --blocks T2` is called and T1 is already in T2's `blocked_by`, T1 gets appended a second time, creating a duplicate entry in the JSONL file. This contradicts the spec's `blocked_by` semantics (an array of IDs implying unique blockers) and is inconsistent with `tick dep add`, which explicitly rejects duplicates with "dependency already exists: %s is already blocked by %s" (dep.go:97-101). The `ValidateDependency` call that follows (update.go:189) only checks cycles and child-blocked-by-parent, not duplicates.
  RECOMMENDATION: Add duplicate checking in `applyBlocks` before appending. For each target task, check whether `sourceID` already exists in its `BlockedBy` slice and skip the append if so. Alternatively, add a deduplication check after `applyBlocks` returns.

SUMMARY: All high-severity findings from cycles 1-2 have been resolved (dependency validation on create/update, Unicode arrow, child-blocked-by-parent error message, create output relationship context). One medium-severity spec conformance gap remains: the --blocks flag on update can create duplicate blocked_by entries, inconsistent with dep add's explicit duplicate rejection.
