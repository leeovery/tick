# Consolidation Findings: same-second-tasks-sort-by-id (Phase 3)

## Findings

None. The phase adds one check (internal/doctor/duplicate_seq.go), one registration (internal/cli/doctor.go:30) and two README sentences with prose assertions (internal/cli/readme_samples_test.go:674-735). DuplicateSeqCheck follows DuplicateIdCheck's shape: group by key in first-appearance order, then report "id (line N)" per group. The two copies do not clear the duplication floor, because a drift between them would only change report wording. The check's "carries a sequence" rule (`seq != 0` on the raw value, internal/doctor/duplicate_seq.go:44) matches storage's backfill rule (`Seq == 0`, internal/storage/jsonl.go:126). Negative and null values fall on the same side in both, so the check cannot warn on a group that backfill would renumber. The `readmeProse` extraction left no copy of the fence-skipping loop behind. go vet and golangci-lint report 0 issues, and the phase's tests pass.

## Comment Corrections

- internal/cli/doctor.go:11-17: this phase updated the count and enumeration. Both restate the `Register` calls directly below, and a count is a cardinality claim that goes stale with every additive check (it already went stale once, from 10 to 11). Only the exit-code and read-only contract carries anything.
  OLD: // RunDoctor executes the doctor diagnostic command. It creates a DiagnosticRunner,
// registers all 11 checks (CacheStalenessCheck, JsonlSyntaxCheck, IdFormatCheck,
// DuplicateIdCheck, OrphanedParentCheck, OrphanedDependencyCheck, SelfReferentialDepCheck,
// DependencyCycleCheck, ChildBlockedByParentCheck, ParentDoneWithOpenChildrenCheck,
// DuplicateSeqCheck),
// runs all checks, formats the output to stdout, and returns the appropriate exit code.
// Doctor is read-only and never modifies data.
  NEW: // RunDoctor executes the doctor diagnostic command, writing the report to
// stdout and returning the process exit code. Doctor is read-only and never
// modifies data.
- internal/doctor/duplicate_seq.go:15-19: the result-shape sentence repeats the `Run` doc four lines below ("one SeverityWarning result per shared sequence"), so both copies have to change together.
  OLD: // DuplicateSeqCheck warns when more than one record in tasks.jsonl carries the
// same creation sequence. Each shared value is reported as its own warning
// naming the tasks and their line numbers. Records with an absent or zero
// sequence carry none and are not compared. It is read-only and never
// modifies the file.
  NEW: // DuplicateSeqCheck warns when more than one record in tasks.jsonl carries the
// same creation sequence. Records with an absent or zero sequence carry none
// and are not compared. It is read-only and never modifies the file.
- internal/cli/doctor_test.go:540-542: this phase's eleventh check falsified "all 10 checks" (the phase's new subtest at :891 runs the fixture against 11). The enumeration restates the three-line fixture below it.
  OLD: // healthyTenCheckContent returns tasks.jsonl content that passes all 10 checks:
// valid IDs, no duplicates, valid JSON, no orphaned parents/deps, no self-refs,
// no cycles, no child-blocked-by-parent, and no done parent with open children.
  NEW:
- internal/cli/doctor_test.go:765: this phase falsified "all 10" (11 checks now run), and the comment restates the subtest name on the line above.
  OLD: // Stale cache (first check fails), but all 10 should still run.
  NEW:
