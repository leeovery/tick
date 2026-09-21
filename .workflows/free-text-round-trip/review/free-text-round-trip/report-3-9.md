TASK: free-text-round-trip-3-9 (tick-98c333) — Corrections: delete a superseded dep-tree test guard and route both dep-tree TOON documents through the shared section join

ACCEPTANCE CRITERIA:
- [x] `grep -rn 'strings\.Join(sections' internal/ cmd/` returns nothing: every TOON document assembled from a `sections` slice is joined by `joinToonSections`.
- [x] Neither deleted subtest remains, and no test in `internal/cli/dep_tree_test.go` asserts only that output differs from `No dependencies found.` or `No dependencies.`.
- [x] Output is byte-identical on every case the suite exercises, in all three formats and both dep-tree modes — the encoder-error path is the only input whose rendering changes, and no test reaches it. (settled by reading of the changed code path; the suite-wide confirmation is criterion 4, recorded under UNSETTLED)
- [ ] `go test ./...` green with no test edits beyond the two deletions, `go vet ./...` clean, `gofmt -l ./internal` empty. (UNSETTLED — requires execution)

STATUS: complete

SPEC CONTEXT: The specification's thesis is that "every section the formatter assembles by hand is malformed; every section it hands to the TOON library is correct" (specification.md:13), and §11 requires the output stay conformant afterwards. The dep-tree documents are directly in that frame: §5 (specification.md:124, :149) converts the chains/longest/blocked summary and the focused target's identity line into top-level named fields, and §8 (specification.md:322) removes the `No dependencies found.` / `No dependencies.` prose from toon and JSON while pretty keeps it. A TOON document that emits a bare blank line with nothing on one side of it because a field section failed to encode is exactly the malformed-document class this work exists to remove, so routing both dep-tree documents through the empty-dropping join is on-thesis rather than cosmetic.

IMPLEMENTATION:
- Status: Implemented (commit d38deefa, `internal/cli/dep_tree_test.go` -48, `internal/cli/toon_formatter.go` 2 lines changed)
- Location:
  - `internal/cli/toon_formatter.go:181-191` (`formatFullDepTree`) and `:193-206` (`formatFocusedDepTree`) — both now end in `doc.join(...)`, which returns `joinToonSections(d.sections)` at `internal/cli/toon_formatter.go:421`. The commit replaced the two `strings.Join(sections, "\n\n")` calls with `joinToonSections(sections)`; a later phase folded the slice into the `toonDoc` collector (`internal/cli/toon_formatter.go:398-422`), preserving the change rather than reverting it. All four TOON documents built from a section slice (`:74`, `:111`, `:183`, `:196`) now reach `joinToonSections`.
  - `internal/cli/dep_tree_test.go` — the subtests `"it outputs dep tree for project with dependencies"` and `"it outputs focused view for task with dependencies"` are gone (enumerated all 45 `t.Run(` calls in the file; neither name is present, and neither name appears anywhere in `internal/cli/`).
- Notes: The literal grep in criterion 1 now returns one hit, `internal/cli/pretty_formatter.go:159` — `strings.Join(sections.Tags, ", ")`. That is a different `sections`: a `detail.selectedSections()` struct in the pretty formatter, joining tag labels with `", "`, not a TOON document assembled from a slice. `git log -S` places its introduction in commit add91a8d (task 10-1), seven phases after this task. The criterion's substance — every TOON `sections` slice joined by `joinToonSections` — holds exactly; only the regex is now shadowed by an unrelated later variable name. Nothing is wrong and no finding follows.

TESTS:
- Status: Adequate (no new test, as the task specifies)
- Coverage: The branches the deleted subtests nominally guarded are pinned by tests that survive unedited:
  - `"it still renders the populated graph"` (`internal/cli/dep_tree_test.go:731-746`) decodes the full-graph TOON document and asserts both edge rows and all three named fields — it fails if the handler takes the no-dependencies path, and unlike the deleted test it also fails if the output is empty.
  - `"it still prints the no-dependencies sentence in pretty for an empty project"` (`:313`) and its sibling (`:326`) assert pretty's stdout equals `"No dependencies found.\n"` byte for byte.
  - `"it decodes dep tree output for a task with both directions"` (`internal/cli/toon_decode_test.go:411-423`) covers the populated focused document end to end: id/title/status fields plus both edge sections.
- Notes: Criterion 2's second half holds. Every remaining reference to the two sentences in `internal/cli/dep_tree_test.go` is a positive pin, not a negative one: `:321` and `:334` assert equality with `"No dependencies found.\n"`, and `:771` asserts the focused pretty output contains `"No dependencies."`. No surviving test in the file asserts only that output differs from either sentence. The deletion leaves no unused import — `strings` (10 uses) and `time` (37 uses) both remain live in the file.

CODE QUALITY:
- Project conventions: Followed. The swap uses the file's existing helper instead of a second hand-rolled join, which is the same DRY move the rest of the formatter already makes for `FormatStats` and `FormatTaskDetail`.
- SOLID principles: Good — single shared join owns the empty-section rule for all four TOON documents.
- Complexity: Low. Two call-site substitutions.
- Modern idioms: Yes.
- Readability: Good.
- Comment accuracy: The false comment the task named (`FormatDepTree` described as "currently a stub returning empty") went with the deleted subtest. The comments remaining on the changed code — `joinToonSections` at `internal/cli/toon_formatter.go:285` and the `toonDoc.join` doc comment at `:415-416` — describe the code accurately and name no process artifact.
- Issues: None.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...` green with no test edits beyond the two deletions, `go vet ./...` clean, `gofmt -l ./internal` empty." — requires executing the three commands. Reading confirms the scope half: commit d38deefa touches exactly two files and its only test edit is the two subtest deletions.
- "Output is byte-identical on every case the suite exercises, in all three formats and both dep-tree modes" — the changed code path is settled by reading (`joinToonSections` differs from `strings.Join(sections, "\n\n")` only when a section is the empty string; `buildEdgeSection` always returns at least a count-zero header, and `encodeToonFields` returned `""` only on a `toon.MarshalString` error over string/int fields, which no suite input produces). Suite-wide byte identity across pretty and JSON would be settled by the criterion-4 run.
