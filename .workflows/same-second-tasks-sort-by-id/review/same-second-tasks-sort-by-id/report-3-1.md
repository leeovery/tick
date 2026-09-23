TASK: Detect Duplicate Creation Sequences In A Read-Only Doctor Check (same-second-tasks-sort-by-id-3-1, tick-deafbb)

ACCEPTANCE CRITERIA:
- A tasks.jsonl whose records on lines 2 and 4 carry the same non-zero seq, with every other record carrying a distinct one: the check returns one failing result at warning severity, naming both tasks and giving lines 2 and 4 (§5.2, §8.3)
- A file in which three records carry one seq: the check returns a single failing result for that group, naming all three tasks with their line numbers (§5.2)
- A file carrying two separate shared values, two records at seq 3 and two at seq 7: the check returns two failing results, one per group, both at warning severity, each giving only its own members' line numbers (§5.2, §8.3)
- A file whose records all carry distinct seq values: the check returns exactly one result, and it passes (§5.2, §8.3)
- A file in which no record carries a seq field, as in a project that predates the field: the check returns exactly one result, and it passes (§5.2)
- A file with two records at seq 0 and two records with no seq field, alongside numbered records carrying distinct values: the check returns exactly one result, and it passes. Records carrying no sequence are compared neither with one another nor with numbered records (§2.2, §5.2)
- A file carrying a duplicate seq: after the check runs, tasks.jsonl is byte-identical to before (§5.2)

STATUS: complete

SPEC CONTEXT: §5.2 calls for a doctor check mirroring DuplicateIdCheck: read-only, one result per duplicate group with its line numbers, a single passing result on a clean file, at SeverityWarning (a duplicate sequence breaks nothing because order stays total via the task-ID final sort term, §5.1). §2.2 says an absent or zero seq means the record carries no sequence. §2.3 backfill numbers unnumbered records above the file's highest, so they cannot collide and are skipped by the check. The doctor reads stored lines (ScanJSONLines), not the backfilled task set. §8.3 asks that the tests mirror DuplicateIdCheck's and that the check report nothing on a clean file. Registration (Task 3-2) and README (Task 3-3) are out of this task.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/doctor/duplicate_seq.go:24-82 (DuplicateSeqCheck.Run); internal/doctor/parent_done_open_children.go:10-13 (stale "only warning-severity check" doc claim removed, as the task's Context anticipated)
- Notes: Structure matches DuplicateIdCheck (internal/doctor/duplicate_id.go:25-92): getJSONLines, skips unparsed lines, groups by key with first-appearance ordering, one failing result per group of two or more, single pass otherwise, and fileNotFoundResult on a missing file. The seq key is read as float64 from the generic JSON map, and non-numeric or zero values are skipped (duplicate_seq.go:41-43). That matches storage, where Task.Seq is an int (internal/task/task.go:57) and only zero is backfilled (internal/storage/jsonl.go:126), so what the doctor treats as "no sequence" is exactly what storage treats that way. null decodes as nil and is skipped, and storage leaves it zero and backfills it, so the two agree there too. Failing results carry SeverityWarning (duplicate_seq.go:68). The Details text names every member's ID and line number, which is what §5.2 asks the report to convey. The check only opens the file for reading via ScanJSONLines. Doc comments on the type and Run (duplicate_seq.go:15-23) and the inline comment at :40 hold true against the code.

TESTS:
- Status: Adequate
- Coverage: Each acceptance criterion has a subtest in internal/doctor/duplicate_seq_test.go. The subtests are: lines 2 and 4 sharing seq 2 among distinct values (:37), three records sharing seq 5 (:55), seq 3 and seq 7 groups returned as two results with only their own lines (:72), all distinct (:95), no seq field (:103), zero and absent seq mixed with distinct numbered records (:111), and byte-identical file after running on a duplicate (:154, via assertReadOnly). The assertions compare whole CheckResult structs (name, Passed, Severity, Details, Suggestion), so a wrong severity, a missing or extra line number, or a group leaking members would each fail. Supporting cases mirror duplicate_id_test.go: empty file (:122), invalid JSON skipped (:126, whose unparseable line embeds "seq":1 so a non-JSON scan would falsely report a duplicate), and a missing file (:134).
- Notes: None. The set is focused, and each subtest covers a distinct condition.

CODE QUALITY:
- Project conventions: Followed. Stdlib testing, t.Run subtests in "it does X" form, t.Helper on the closures, the shared doctor test helpers (setupTickDir, writeJSONL, ctxWithTickDir and assertReadOnly in internal/doctor/cache_staleness_test.go), and the fileNotFoundResult helper.
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes
- Readability: Good
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
