TASK: Run The Duplicate-Sequence Check From Tick Doctor (same-second-tasks-sort-by-id-3-2, tick-1867d8)

ACCEPTANCE CRITERIA:
- A project with a fresh cache whose `tasks.jsonl` is healthy apart from two records sharing a non-zero `seq`: `tick doctor` prints the duplicate-sequence check as failing, naming both tasks with their line numbers, and exits 0 (§5.2, §8.3)
- A healthy project whose records carry distinct `seq` values: `tick doctor` prints the duplicate-sequence check as one passing line alongside the ten existing checks, and exits 0 (§5.2)
- A healthy project whose records carry no `seq` field, as one predating the field: `tick doctor` prints the duplicate-sequence check as one passing line, and exits 0 (§5.2)
- A `tasks.jsonl` in the merge shape, every record in one creation second, in line order: `a`, `b`, `c` carrying `seq` 1, 2 and 3; `x` and `y` carrying none; then `d` and `e` carrying 4 and 5. After the next write through tick, the rewritten file gives `x` and `y` `seq` 6 and 7, `a` through `e` keep 1 to 5, no `seq` value appears twice, and `tick doctor` prints the duplicate-sequence check passing (§2.3, §8.2)
- `RunDoctor`'s doc comment names the duplicate-sequence check among the registered checks and gives the count as 11 (§5.2)

STATUS: issues_found

SPEC CONTEXT: §5.2 adds a doctor check that reports records sharing a creation sequence, naming the tasks and their lines, at warning severity so `tick doctor` still exits zero (a failing run would leave a project permanently red after a routine merge, since tick has no command that renumbers a sequence). It registers in `RunDoctor`. §2.3 (as corrected by the 2026-09-22 corrigendum) numbers unnumbered records above the highest sequence the file carries, never with a running maximum; §8.2 requires the merge shape to carry no duplicate and the doctor to report clean after the next write. §8 fixture constraint: records share one creation second where the sequence is under test.

IMPLEMENTATION:
- Status: Implemented (AC5 deliberately drifted, see notes)
- Location: internal/cli/doctor.go:26 (registration of `doctor.DuplicateSeqCheck`, last in the block after `ParentDoneWithOpenChildrenCheck`); internal/doctor/duplicate_seq.go:24-82 (the check, from Task 3-1: SeverityWarning at :68, "Sequence uniqueness" name at :25, "id (line N)" details at :62/:69); internal/doctor/format.go (`ExitCode` returns 1 only on error-severity failures, so a warning leaves exit 0); internal/storage/jsonl.go:123-131 and internal/storage/store.go:192 (backfill above the file's highest sequence, which the merge-shape scenario exercises through a write).
- Notes: AC1-AC4 hold on reading: the check is registered and runs over the pre-scanned lines; the report prints "✗ Sequence uniqueness: Duplicate sequence N: <id> (line L), ..." with the suggestion line and the run exits 0; records without `seq` are skipped so legacy files pass; `Store.Mutate` backfills and marshals, so the merge shape is written with x=6, y=7. AC5 is deliberately not met as worded: commit a6f49a4f grew the enumeration to 11 and named `DuplicateSeqCheck`, then the phase-3 comment pass (4c15eb60) replaced the whole enumeration with "writing the report to stdout and returning the process exit code. Doctor is read-only and never modifies data." The analysis records this as a deliberate choice (implementation/.../analysis-standards-c1.md:18, consolidation-findings-p3.md): a list of names and a count restating the `Register` calls directly below goes stale with every added check. The spec asked for the comment to update so it would stay true. The replacement stays true and cannot drift, so nothing the intent needs is lost, and a comment-only remedy could never block anyway. Not reported.

TESTS:
- Status: Adequate
- Coverage: internal/cli/doctor_test.go:850 (AC1: fresh-cache fixture, exact failing line plus suggestion, exit 0, 10 other check marks, "1 issue found." summary); :873 (AC2: distinct seqs, exactly one passing line, no failure line, exit 0, 11 check marks); :887 (AC3: `healthyTenCheckContent`, no `seq`, one passing line, exit 0); :893 (AC4: `mergeShapeLines()` fixture from backfill_seq_test.go:97-107, all records at `sameSecond`, write via `update`, raw on-disk read via `rawRecords` asserting every stored seq against `mergeShapeSeqs` (x=6, y=7, a-e 1-5) plus a no-duplicate map, then doctor passes). The two existing count tests (:320-329, :566-575) were updated to 11. Each test fails if its behaviour breaks: removing the registration breaks :850/:873/:887 and both counts, error severity breaks :850's exit code, and a running-maximum backfill breaks :893 on both the stored values and the doctor pass. The backfill tests (backfill_seq_test.go:191, :241) do not duplicate :893, which adds what they lack: the written file and the doctor verdict.
- Notes: :873 runs the doctor twice (once inside `assertSeqCheckPasses`, once for the count). That is redundant but breaks nothing. See FINDINGS for the stale "10 checks" subtest names in `TestDoctorTenChecks`.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, `t.Run` "it ..." subtests, `t.Helper()` on `assertSeqCheckPasses`, `t.TempDir()` isolation through the setup helpers)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes
- Readability: Good
- Issues: Subtest names in `TestDoctorTenChecks` still claim a 10-check doctor (see FINDINGS)

BLOCKING ISSUES:
- None

FINDINGS:
- [in-scope] [contained] internal/cli/doctor_test.go:761 — Registering the eleventh check made the "10 checks" claims in `TestDoctorTenChecks` false, and this task left them in place: "all 10 checks" in the subtest names at :554, :577, :721, :761 and :773; "with 10 checks" at :788 and in the failure messages at :815 and :818; "all 10 checks pass" at :822; "(other 9 pass)" at :587, :598, :609, :620 and :632, where 10 other checks now pass. The `allLabels` list at :547-552 is iterated as every check's label by :554, :721, :761 and :773, and it omits "Sequence uniqueness". Fix: add "Sequence uniqueness" to `allLabels`, and change the counts in those names and messages to 11 and "other 10" (or drop the counts). The same phase's comment pass (4c15eb60) deleted two comments in this file for making this same false "all 10" claim, the `healthyTenCheckContent` doc and the comment inside :761, but it left the subtest names that make it. — FAILS: the suite describes a 10-check doctor that no longer exists. The no-short-circuit subtest at :761 checks only 10 of the 11 labels, so it would still pass if the sequence check stopped running after the stale-cache failure.

UNSETTLED:
- None
