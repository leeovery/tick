# Phase 3: Duplicate-sequence visibility and the documented contract — 3 tasks

## same-second-tasks-sort-by-id-3-1

### Task 3-1: Detect duplicate creation sequences in a read-only doctor check

**Problem**: A duplicate creation sequence can happen in this project. `.tick/tasks.jsonl` is git-tracked (`.gitignore` excludes only `cache.db`, `lock` and temp files), and implementation runs in worktrees on branches. Two branches that each number new tasks from the same highest sequence merge into records that share one. Where those tasks also record the same creation second, the group loses its authoring order and falls back to task-ID order among themselves. The order is defined but wrong, and nothing says so: seven children forced to one sequence returned the wrong order with no warning. Tick offers no command that renumbers a sequence, so the only repair is a hand edit of `tasks.jsonl`, and the user needs to know which tasks share a number and on which lines to make it.

**Solution**: Add a doctor check that mirrors `DuplicateIdCheck`. It groups the records in `tasks.jsonl` by their `seq` value and reports every value that more than one record carries, naming the tasks and their line numbers, at warning severity. Records that carry no sequence (absent or zero) take no part in the comparison. The check only reads the file.

**Outcome**: Run against a `tasks.jsonl`, the check returns one warning per shared sequence value, each naming the tasks that share it and the lines they sit on. With no shared value it returns a single passing result. A project that predates the field reports clean, and the file is never modified.

**Acceptance Criteria**:
- [ ] A `tasks.jsonl` whose records on lines 2 and 4 carry the same non-zero `seq`, with every other record carrying a distinct one: the check returns one failing result at warning severity, naming both tasks and giving lines 2 and 4 (§5.2, §8.3)
- [ ] A file in which three records carry one `seq`: the check returns a single failing result for that group, naming all three tasks with their line numbers (§5.2)
- [ ] A file carrying two separate shared values, two records at `seq` 3 and two at `seq` 7: the check returns two failing results, one per group, both at warning severity, each giving only its own members' line numbers (§5.2, §8.3)
- [ ] A file whose records all carry distinct `seq` values: the check returns exactly one result, and it passes (§5.2, §8.3)
- [ ] A file in which no record carries a `seq` field, as in a project that predates the field: the check returns exactly one result, and it passes (§5.2)
- [ ] A file with two records at `seq` 0 and two records with no `seq` field, alongside numbered records carrying distinct values: the check returns exactly one result, and it passes. Records carrying no sequence are compared neither with one another nor with numbered records (§2.2, §5.2)
- [ ] A file carrying a duplicate `seq`: after the check runs, `tasks.jsonl` is byte-identical to before (§5.2)

**Do**:
- Add a new check in `internal/doctor/` that mirrors `DuplicateIdCheck` (`internal/doctor/duplicate_id.go`): read-only, one failing result per duplicate group carrying that group's line numbers, a single passing result on a clean file, and records with no value for the compared key skipped. Its tests mirror `DuplicateIdCheck`'s existing tests (`internal/doctor/duplicate_id_test.go`) (§5.2, §8.3).
- The compared value is the record's `seq` field (§3.1). An absent or zero value means the record carries no sequence (§2.2).
- Failing results carry `SeverityWarning` (`internal/doctor/doctor.go`), the severity `ParentDoneWithOpenChildrenCheck` reports at (`internal/doctor/parent_done_open_children.go:59`) (§5.2).

**Context**:
> §5.2: "A new check mirrors `DuplicateIdCheck` (`internal/doctor/duplicate_id.go`): read-only, never modifies the file, reports each duplicate group with its line numbers and returns a single passing result on a clean file. Records carrying no sequence — absent or zero (§2.2) — are not compared with one another: backfill numbers each of them above every sequence the file carries (§2.3), so they cannot collide with one another or with a carried sequence, and `DuplicateIdCheck` already skips records with no ID. A project that predates the field reports clean."
>
> §5.2 on severity: "The report does not fail the run. … a duplicate sequence breaks nothing: order stays total (§5.1) and the group falls back to ID order among themselves. It reports at warning severity, joining `ParentDoneWithOpenChildrenCheck` …, the one existing warning against nine errors. Failing the run would leave a project permanently red to any script or CI gate after a routine branch merge, clearable only by hand, since tick offers no command that renumbers a sequence." `SeverityWarning` is "a suspicious but allowed state that does not affect exit code" (`internal/doctor/doctor.go:13-17`).
>
> §5.2 on what the report conveys: "What the report tells the user is which tasks share a number and on which lines. Sharing a number costs the authoring order only where the tasks also record the same creation second — there the group falls back to ID order among themselves; where their recorded creation dates differ, those dates decide and the listed order is unaffected (§4.2). Where the order was lost, restoring it means editing the sequences in `tasks.jsonl` so they differ." Every duplicate group is reported, whatever creation seconds its members record.
>
> §5 / §5.2: a duplicate-identity condition gets a doctor check, not a hard refusal on read. "A refusal would block every command after a merge; detection plus a defined fallback gives both properties." The defined fallback, the task-ID final sort term, is Phase 1.
>
> §2.2: "Sequences are positive: numbering begins at 1, so an absent or zero value on a record means it carries no sequence."
>
> The doctor reads the stored lines (`ScanJSONLines`, `internal/doctor/jsonl_reader.go`), not the task set after backfill, so it sees each record exactly as stored. A record written with no sequence still has none when the doctor reads it, even though the storage layer numbers it on every read. That is why unnumbered records are skipped rather than compared.
>
> `ParentDoneWithOpenChildrenCheck`'s doc comment (`internal/doctor/parent_done_open_children.go:11-13`) calls it "the only warning-severity check in the doctor suite". With this check added, that is no longer so.
>
> The `seq` field and its backfill are Phase 1. Registering the check in `tick doctor` is Task 3-2, and naming it in the README is Task 3-3.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §2.2, §2.3, §3.1, §5, §5.1, §5.2, §8.3

## same-second-tasks-sort-by-id-3-2

### Task 3-2: Run the duplicate-sequence check from tick doctor

**Problem**: A check the runner never registers never runs. `RunDoctor` (`internal/cli/doctor.go:17`) registers every check explicitly, and `tick doctor` is where a user looks after a merge, the one event that makes a duplicate sequence reachable. It has to report there without failing the run: a failing run would turn every routine branch merge red for any script or CI gate, clearable only by hand, since tick has no command that renumbers a sequence. The same merge can also leave records with no sequence lying above numbered ones. Were the backfill to number those into values already carried, tick would manufacture duplicates itself, silently, until the next write froze them into the file. That merge shape needs an end-to-end assertion through the doctor.

**Solution**: Register the duplicate-sequence check in `RunDoctor` alongside the ten existing checks, and update the doc comment that enumerates them. Assert through `tick doctor` that a duplicate reports as a warning and the run exits zero, that a clean project and one predating the field show the check passing, and that a merge-shaped file carries no duplicate once the next write has made its backfilled sequences permanent.

**Outcome**: `tick doctor` runs eleven checks. On a file carrying a duplicate sequence it prints the group with its line numbers and exits zero. On a clean project, or one that predates the field, the new check shows passing. A merge-shaped file, once written, carries no duplicate sequence and `tick doctor` reports the check clean.

**Acceptance Criteria**:
- [ ] A project with a fresh cache whose `tasks.jsonl` is healthy apart from two records sharing a non-zero `seq`: `tick doctor` prints the duplicate-sequence check as failing, naming both tasks with their line numbers, and exits 0 (§5.2, §8.3)
- [ ] A healthy project whose records carry distinct `seq` values: `tick doctor` prints the duplicate-sequence check as one passing line alongside the ten existing checks, and exits 0 (§5.2)
- [ ] A healthy project whose records carry no `seq` field, as one predating the field: `tick doctor` prints the duplicate-sequence check as one passing line, and exits 0 (§5.2)
- [ ] A `tasks.jsonl` in the merge shape, every record in one creation second, in line order: `a`, `b`, `c` carrying `seq` 1, 2 and 3; `x` and `y` carrying none; then `d` and `e` carrying 4 and 5. After the next write through tick, the rewritten file gives `x` and `y` `seq` 6 and 7, `a` through `e` keep 1 to 5, no `seq` value appears twice, and `tick doctor` prints the duplicate-sequence check passing (§2.3, §8.2)
- [ ] `RunDoctor`'s doc comment names the duplicate-sequence check among the registered checks and gives the count as 11 (§5.2)

**Do**:
- `internal/cli/doctor.go`: register the new check in `RunDoctor` alongside the others (§5.2 cites `internal/cli/doctor.go:22`, inside the registration block).
- Update the doc comment above `RunDoctor` (lines 11–16), which enumerates the registered checks by name and count ("registers all 10 checks"), so it names the new check and gives the count as 11 (§5.2).

**Context**:
> §5.2: "It registers alongside the others in `RunDoctor` (`internal/cli/doctor.go:22`), whose doc comment enumerates the registered checks by name and count (… "registers all 10 checks") and updates with the addition."
>
> §5.2 / §8.3: "A duplicate does not fail the run: the check reports at warning severity and `tick doctor` still exits zero on a file carrying one." `doctor.ExitCode` (`internal/doctor/format.go`) returns 1 only for failures at error severity. `FormatReport` prints warnings and errors alike as `✗` lines and counts both in its "issues found" summary.
>
> §2.3, the merge shape: "a branch cut before the field existed appends `x` and `y` with no sequence; an upgraded `main` rewrites `a`, `b`, `c` as 1, 2, 3 and appends `d` (4) and `e` (5); resolving the conflict at the end of the file can leave `x` and `y` above `d` and `e`. A running maximum numbers them 4 and 5 — duplicates tick would manufacture itself, silent until the next write froze them and the doctor check (§5.2) reported a hand-edit chore. Numbering above the file's highest gives them 6 and 7, so a backfilled number can never equal a carried one." §8.2: this file "backfills with no sequence equal to any carried one, and the doctor check reports clean after the next write. This is the case a running-maximum rule gets wrong."
>
> The backfill rule under test is §2.3 as corrected by the specification's 2026-09-22 corrigendum: each record with no sequence, taken in line order, gets the next number above the highest sequence the file carries, which is new-task numbering (§2.2) applied per record. Backfill is Phase 1 work, and this scenario checks its outcome through the doctor. The assertion comes after the write because, before it, `x` and `y` carry no sequence in the stored bytes and the check skips records with no sequence, so the doctor would pass whatever the backfill did. Only the write puts a colliding backfill into the file.
>
> §8 fixture constraints: creation seconds tie wherever the sequence is under test, so the merge-shape records share one creation second.
>
> Two existing tests count the passing check marks on a healthy project and expect 10: "it runs all four checks in a single tick doctor invocation" in `TestDoctorFourChecks` (`internal/cli/doctor_test.go:320`) and "it runs all 10 checks in a single tick doctor invocation" in `TestDoctorTenChecks` (`:570`). Once the new check is registered, a healthy project prints 11. Their fixtures carry no `seq`, so the new check passes on them.
>
> `doctor` is out of scope for the conformance inventory (`conformanceOutOfScopeCommands`, `internal/cli/conformance_test.go:1595`), and the README has no `$ tick doctor` sample, so neither changes.
>
> The check itself is Task 3-1. The README's doctor enumeration is Task 3-3.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §2.2, §2.3, §5.2, §8 (fixture constraints), §8.2, §8.3

## same-second-tasks-sort-by-id-3-3

### Task 3-3: Document the creation-order tiebreak and the duplicate-sequence check in the README

**Problem**: `README.md:115` is the only place that states the sort contract. It promises "sorted by priority (ascending), then creation date" and says nothing about tasks whose creation dates are equal, so today neither the code nor the docs commit to an answer. With the sort ending on the creation sequence, that tiebreak is now a guarantee, and because the sequence is never printed (§2.4) the documentation is the only place a user can learn it. `README.md:396` lists what `tick doctor` checks for, and its "duplicates" reads as duplicate IDs, so a user whose tasks lost their authoring order after a merge cannot find the diagnostic that reports it. Both sentences are prose outside every fence. The README-sample run re-renders only fences carrying a `$ tick` prompt line, and nothing in the suite reads the sort sentence today, so either sentence could be deleted or folded back without a test noticing.

**Solution**: Extend the `list` section's sort sentence with the tiebreak: within a priority band, tasks tied on creation date come back in creation order. Add duplicate creation sequences to the `doctor` section's enumeration as an entry of its own. Pin each sentence with a prose assertion, following the two the suite already carries for README prose.

**Outcome**: The README states the creation-order tiebreak and names the duplicate-sequence diagnostic. Removing the tiebreak, or folding the doctor entry back into "duplicates", fails a test. The README sample run and `tick help` output are unchanged.

**Acceptance Criteria**:
- [ ] The README's `list` section states that results are sorted by priority (ascending), then creation date, and that within a priority band tasks tied on creation date come back in creation order (§1.3, §6)
- [ ] With the tiebreak removed from that section's prose, the new sort-contract prose assertion fails; with it present, the assertion passes (§8.6)
- [ ] The README's `doctor` section lists duplicate creation sequences among the things `tick doctor` checks for, as an entry of its own joining the existing ones (§6)
- [ ] With that entry removed, or folded back into the existing "duplicates", the new doctor-enumeration prose assertion fails; with it present, the assertion passes (§8.6)
- [ ] The README sample run passes with no sample, fixture or exemption changed: both sentences sit outside every fence (§6)
- [ ] `tick help list` and `tick help doctor` print exactly what they print today; `internal/cli/help.go` is unchanged (§6)

**Do**:
- `README.md:115`, in the `list` section: add the tiebreak to the sort-contract sentence (§6).
- `README.md:396`, in the `doctor` section: add duplicate creation sequences to the enumeration as an entry of its own (§6).
- Add the two prose assertions following `TestREADMEDocumentsFieldSelection` and `TestREADMEDocumentsEndOfFlagsMarker` (`internal/cli/readme_samples_test.go:570` and `:637`) (§8.6).

**Context**:
> §6: "`README.md:115` is the only site stating the sort contract (`rg -n 'sorted by|creation date' README.md` → one hit). It promises 'sorted by priority (ascending), then creation date' and says nothing about what happens when creation dates are equal, so neither the code nor the docs commit to an answer today. It gains the tiebreak: within a priority band, tasks tied on creation date come back in creation order."
>
> §6: "`README.md:396` … enumerates what `tick doctor` checks for … The duplicate-sequence check (§5.2) joins that enumeration, named as duplicate creation sequences rather than folded into the existing 'duplicates', which reads as duplicate IDs — a user whose tasks lost their authoring order after a merge has to be able to find the diagnostic that reports it. The sentence is prose outside every fence, so no README sample renders it."
>
> §6 on `internal/cli/help.go`: it "contains no statement about sort order at all, and needs no change". `tick help doctor`'s description (`internal/cli/help.go:217`) lists checks "as a summary, not a standing enumeration: it already omits self-referential dependencies, parent/child constraint violations and a done parent with open children". The rule that puts the new check in the README's list, that a standing list of every check either gains it or is wrong, does not reach a list that makes no claim to completeness, so that description stays unchanged (settled by the specification's 2026-09-22 corrigendum).
>
> §6 / §8.6: the README-sample run re-renders only fenced blocks carrying a `$ tick` prompt line (`internal/cli/readme_samples_test.go:492`, `if fence.prompt == "" {`), and nothing in the suite reads the sort sentence today (`rg -n 'sorted by|priority \(ascending\)' internal/cli/*_test.go` → no output). The sort-contract sentence "is the only user-facing statement of the guarantee this work delivers, and the sample run never reads it". The doctor enumeration "is the only place a user meets the new diagnostic, it is prose the sample run never reads, and folding it back into the generic 'duplicates' would otherwise pass unnoticed."
>
> §1.3, the guarantee: "Tasks come back in the order they were created, within a priority band — and within the `in_progress` band for `ready`, which floats above the priority terms." A mixed-priority batch still does not read back in write order: priority and the `ready` band outrank creation order.
>
> §2.4: the sequence is "an ordering mechanism, not information about the work". It appears nowhere a command prints and gets no README sample. "The guarantee it delivers is stated in the documentation (§6)."
>
> The guarantee holds once Phase 1's list-family sort ends on created, sequence, task ID. The diagnostic the doctor entry names is built in Tasks 3-1 and 3-2.

**Spec Reference**: `.workflows/same-second-tasks-sort-by-id/specification/same-second-tasks-sort-by-id/specification.md` — §1.3, §2.4, §5.2, §6, §8.6
