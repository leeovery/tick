TASK: Document The Creation-Order Tiebreak And The Duplicate-Sequence Check In The README (same-second-tasks-sort-by-id-3-3, tick-1e70a2)

ACCEPTANCE CRITERIA:
- The README's `list` section states that results are sorted by priority (ascending), then creation date, and that within a priority band tasks tied on creation date come back in creation order (§1.3, §6)
- With the tiebreak removed from that section's prose, the new sort-contract prose assertion fails; with it present, the assertion passes (§8.6)
- The README's `doctor` section lists duplicate creation sequences among the things `tick doctor` checks for, as an entry of its own joining the existing ones (§6)
- With that entry removed, or folded back into the existing "duplicates", the new doctor-enumeration prose assertion fails; with it present, the assertion passes (§8.6)
- The README sample run passes with no sample, fixture or exemption changed: both sentences sit outside every fence (§6)
- `tick help list` and `tick help doctor` print exactly what they print today; `internal/cli/help.go` is unchanged (§6)

STATUS: complete

SPEC CONTEXT: §6 names README.md:115 as the only statement of the sort contract and prescribes the tiebreak wording ("within a priority band, tasks tied on creation date come back in creation order"); it adds the duplicate-sequence check (§5.2) to the README's doctor enumeration as "duplicate creation sequences", distinct from the existing "duplicates" (which reads as duplicate IDs). §2.4 keeps the sequence out of all command output, so the README is the only place the guarantee is stated. The 2026-09-22 corrigendum settles that `tick help doctor` (help.go:217) is a summary and stays unchanged. §8.6 requires each sentence be pinned by a prose assertion modelled on TestREADMEDocumentsFieldSelection and TestREADMEDocumentsEndOfFlagsMarker.

IMPLEMENTATION:
- Status: Implemented
- Location: README.md:115 (sort contract plus tiebreak), README.md:396 (doctor enumeration with "duplicate creation sequences" as its own entry after "duplicates")
- Notes: README.md:115 reads "Results are sorted by priority (ascending), then creation date; within a priority band, tasks tied on creation date come back in creation order.", which is the spec's prescribed wording verbatim. README.md:396 lists "duplicates, duplicate creation sequences" as separate comma-separated entries. Both are plain prose lines, neither starts with "```" and neither sits inside a fence, so fencesIn (internal/cli/readme_samples_test.go:70) never collects them and no prompted sample, fixture or exemption can be affected by the edit. The seeded sample fixtures (readme_samples_test.go:268-335) carry no seq fields, the one exemption (:464) is unchanged in shape, and the two anchor tests still sit at :570 and :637, the line numbers the pre-implementation spec cited, so nothing above them shifted. internal/cli/help.go is not among the files the feature's commits touched; help.go:217 still carries the original doctor summary. The README's list section does not mention the `ready` in_progress band, but that gap predates this work and the spec prescribed the added clause as written.

TESTS:
- Status: Adequate
- Coverage: TestREADMEDocumentsSortContract (internal/cli/readme_samples_test.go:691) takes the prose outside every fence of the `### \`list\`` section (readmeProse at :676, readmeSection at :559) and requires both "sorted by priority (ascending), then creation date" and the full tiebreak clause as substrings. Removing or rewording the tiebreak fails the second subtest. TestREADMEDocumentsDuplicateSequenceCheck (:725) parses the "Checks for:" line into its comma-separated entries (readmeDoctorChecks at :708, which strips the trailing "." and the "and " prefix) and requires both "duplicates" and "duplicate creation sequences" as exact entries. Removing the new entry fails. Folding it into "duplicates" in any form ("duplicates (IDs and creation sequences)", "duplicate IDs and creation sequences", or "duplicates" alone) also fails, because the check is exact-entry membership. Keeping the "duplicates" assertion pins the "joining the existing ones" part of the criterion, so the new entry cannot silently replace the ID check.
- Notes: The tests are small and use the file's existing helpers (readmeSection, readmeProse). Subtest names follow the project's "it ..." convention. Neither test is redundant, and each would fail if its sentence regressed.

CODE QUALITY:
- Project conventions: Followed (stdlib testing only, t.Run subtests, t.Helper on readmeDoctorChecks, t.Fatal plus unreachable return mirrors readmeGlobalFlagLabels)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes (strings.SplitSeq, strings.CutPrefix, slices.Contains)
- Readability: Good
- Issues: None. The doc comments on readmeProse (:674) and readmeDoctorChecks (:706) match what the code does and cite no process artifacts.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "The README sample run passes with no sample, fixture or exemption changed: both sentences sit outside every fence (§6)". Reading settles the fence half: both sentences are outside every fence and no sample, fixture or exemption moved. Whether the sample run passes over the whole change-set (with the Phase 1 sort terms and seq backfill in place) needs `go test ./internal/cli -run 'TestREADME' -count=1` to be run.
