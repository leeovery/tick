# Review Tracking: Same-Second Tasks Sort By ID - Claims Verification

## Findings

### 1. The README sort-contract sentence is guarded by no test

**Source**: Tree measurement — `awk 'BEGIN{inf=0} /^```/{inf=!inf} NR==115{print inf}' README.md`, `sed -n '34p;486,494p' internal/cli/readme_samples_test.go`, `rg -n 'sorted by|creation date' internal/cli/*_test.go`
**Category**: Source defect
**Move**: route
**Affects**: Section 6 (Documentation)

**Problem**:
The README sentence describing how `tick list`, `tick ready` and `tick blocked` order their results is the only place a user can read what order tick returns tasks in, and this work rewrites it to promise that tasks tied on creation date come back in creation order. The record tells the implementer that whatever lands in the README is covered by the byte-for-byte sample run. It is not. That run re-renders only fenced blocks whose first line is a `$ tick …` prompt; the sort-contract sentence is prose sitting outside every fence, and no test in the suite reads it. So the one user-facing statement of the new guarantee can be worded wrong at the moment it lands, or drift silently the next time the sort changes, and the suite stays green either way — a user who scripts against the documented order gets a different one with nothing to warn them.

**Evidence**:
Claim (Section 6, final paragraph): "Whatever lands in the README is covered by the README-sample run (`readme_samples_test.go`), which compares each prompted fence byte-for-byte."

The sentence being changed is prose, not a fence:
```
$ awk 'BEGIN{inf=0} /^```/{inf=!inf} NR==115{print "line 115 inside fence: " (inf?"yes":"no"); print $0}' README.md
line 115 inside fence: no
List tasks with optional filters. Results are sorted by priority (ascending), then creation date.
```

The sample run only claims fences carrying a `$ tick` prompt line — every other fence, and all prose, is skipped:
```
$ sed -n '34p;486,494p' internal/cli/readme_samples_test.go
	readmeShellPromptLinePrefix  = "$ tick"
	var errs []error
	exempted := make(map[string]bool, len(exemptions))
	occurrences := make(map[readmeAnchor]int, len(fences))
	for _, fence := range fences {
		anchor := readmeAnchor{info: fence.info, firstLine: firstLineOf(fence.body)}
		occurrences[anchor]++
		if fence.prompt == "" {
			continue
		}
```

Nothing else in the suite reads the sentence:
```
$ rg -n 'sorted by|creation date|priority \(ascending\)' internal/cli/*_test.go
(no output)
```

The README prose assertions that do exist cover other text — the `show` section's flag list and the Global Flags block:
```
$ rg -n '^func TestREADME' internal/cli/readme_samples_test.go
165:func TestREADMEToonSamplesDecode(t *testing.T) {
210:func TestREADMETransitionSamples(t *testing.T) {
432:func TestREADMESamplesMatchRenderedOutput(t *testing.T) {
521:func TestREADMEPromptedSamplesAreClaimed(t *testing.T) {
570:func TestREADMEDocumentsFieldSelection(t *testing.T) {
637:func TestREADMEDocumentsEndOfFlagsMarker(t *testing.T) {
```

Source carrying the claim: `.workflows/same-second-tasks-sort-by-id/investigation/same-second-tasks-sort-by-id.md`, "Testing Recommendations" → "Documentation and regression floor", line 297: "`README.md:115` and `help.go:58` state the sort contract; whatever lands there is covered by the README-sample run."

**Proposed Text**:

**Resolution**: Pending
**Notes**:

---

## Observations

- Section 3.1 cites `internal/task/task.go:44-60` (the `Task` struct and its JSON tags) as where the record's fields live; the stored record's field set is produced by `taskJSON` (`task.go:62-79`) and the hand-copy in `MarshalJSON`/`UnmarshalJSON` (`task.go:82`, `:106`), so a tag on `Task` alone emits nothing.
- Section 7.1 cites `internal/cli/dep_tree.go:26` for `store.ReadTasks()`; the call is at line 25 (`grep -n 'store.ReadTasks()' internal/cli/dep_tree.go` → `25:`). Carried from the investigation.
- Section 4.4 cites `README.md:206` for the `children.N` / `blocked_by.N` positional form; line 206 is the `$ tick show tick-a1b2 --field notes.2` sample and the children/blocked_by sentence is line 210. Carried from the investigation.
- Section 2.3 names "the first `MarshalJSONL` (commit `4278ba09`)"; that commit's writer is `WriteJSONL` with an inline unsorted `for _, t := range tasks` loop, and `MarshalJSONL` first appears in `23e0dc0f`. The substance — no write path has ever sorted — holds (`git log -S "sort." -- internal/storage/ internal/task/ internal/cli/create.go` → no commits).
- Section 5.2 cites `internal/cli/doctor.go:22` for `RunDoctor`; the function opens at line 17 and line 22 is one of the `runner.Register` calls inside it.
- Section 8 cites `list_filter_test.go:368` beside the other fixtures; the fixture is at lines 349-355 and 368 is an assertion line within the same subtest.
- Section 1.2's "no live exposure" holds for live work: `.tick/tasks.jsonl` carries two same-second groups (2 and 5 tasks), every member `done`.
