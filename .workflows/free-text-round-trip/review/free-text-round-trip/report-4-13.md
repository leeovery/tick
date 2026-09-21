TASK: free-text-round-trip-4-13 (tick-1267e8) — Every README Sample Is Pinned To Real Output

ACCEPTANCE CRITERIA:
- `tick show tick-a1b2 --field description` renders the body of the bare-description fence byte-for-byte from the seeded fixture, and `--field notes.2` renders the body of the bare-note fence.
- Each of the two new samples resolves to exactly one README fence.
- Every `$ tick`-prompted fence in the README is claimed by a sample or by the single declared exemption.
- Adding an unclaimed `$ tick` fence fails the suite with a message naming its prompt line.
- An exemption naming no fence fails the suite.
- README content is unchanged; the diff touches `internal/cli/readme_samples_test.go` only, and `go test ./...` passes.

STATUS: complete

SPEC CONTEXT:
§12.1 ("The README is updated as part of this work") holds that the README is live documentation, so leaving it describing output the tool does not produce is shipping a defect — which is what the byte-for-byte sample harness exists to prevent. §9.7 states that a request returning a bare value ignores `--json`, `--pretty` and `--toon`, while a request returning a document honours them; running both new samples under `--toon` and still matching the unformatted fence bytes is that claim pinned. The two fences this task covers are the bare-value examples §9.2/§9.7 describe.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/cli/readme_samples_test.go:26-27 (anchors), :43-47 (`readmeFence.prompt`), :110-125 (`fenceBody` returning the prompt, `firstLineOf`), :349-360 (the two new sample groups), :454-503 (`readmeSampleExemptions`, `readmeAnchor`, `readmeClaim`, `readmeCoverageErrors`), :505-542 (`allREADMESamples`, `TestREADMEPromptedSamplesAreClaimed`). Commit b9782df7, one file, 122 insertions / 11 deletions.
- Notes:
  - Both new samples use the existing `fieldSelectionTasks` fixture (:311-330), which carries `Description: "Full task description here.\nCan be multiple lines."` and a second note `Blocked on the migration landing` — exactly the fence bodies at README.md:186-190 and :205-208. `RunShow` prints a bare value via `fmt.Fprintln` (internal/cli/show.go:71-76) before any formatter is consulted, and the harness trims trailing newlines (:437), so the comparison is the fence's two lines against the description's own bytes; `--toon` is inert on that path, which is the §9.7 claim the task wanted pinned. `notes.2` resolves through `barePositionValue` → `noteTexts` (internal/cli/show_fields.go:100-106), returning the note text alone.
  - Anchor uniqueness checked by enumerating every fence in README.md: `Full task description here.` is the first line of exactly one fence (186) and `Blocked on the migration landing` of exactly one (205) — it also appears inside the `--field description,notes` fence body but not as its first line, so `occurrence: 1` is right for both and no other sample's occurrence is disturbed.
  - Coverage guard verified against the README as it stands: twelve fences carry a `$ tick` prompt line (README.md:187, 195, 206, 329, 342, 437, 449, 516, 526, 560, 571, 639, each the first line of its fence). Eleven map to a declared sample anchor at occurrence 1 and the twelfth (`$ tick list --stauts open`, README.md:639) matches the one exemption key at :457 exactly. No prompted fence is unclaimed and the exemption is used, so neither arm of `readmeCoverageErrors` fires.
  - Occurrence counting in the guard (:481-492) increments over every fence with a matching info+first-line pair, prompted or not — the same rule `findREADMEBlock` uses (:393-408), so a sample's `occurrence` means the same thing to both.
  - README is untouched: `git show b9782df7 -- README.md` is empty and the commit stat lists only the test file.
  - Plan wording says the guard should assert each prompted fence is claimed by "exactly one" sample; the implementation asserts at least one (a `map[readmeClaim]bool` set, :473-477). Two samples claiming the same fence would go unreported — but both would still have to render it byte-for-byte in `TestREADMESamplesMatchRenderedOutput`, so no wrong sample can hide behind the duplicate. Sound divergence, not a loss.

TESTS:
- Status: Adequate
- Coverage: All five named tests exist and are the behaviour, not the mechanism — "it reproduces the README bare description sample" (:350), "it reproduces the README bare note position sample" (:356), "it claims every prompted README sample" (:515), "it fails when a prompted sample is claimed by nothing" (:521), "it fails when an exemption names no fence" (:532).
- Notes:
  - The two negative tests exercise `readmeCoverageErrors` — the same pure function the live guard calls at :516 — with synthetic fences and exemptions, so the class is checked without editing README.md and then reverting. Each asserts the error count and that the message names the offending prompt line, which is what the acceptance criteria ask of the failure mode.
  - Both would fail if the feature broke: dropping the unclaimed-fence branch leaves `len(errs) == 0` in the first, dropping the unused-exemption loop the same in the second.
  - Not over-tested: one sample entry per new fence, no redundant assertions, no fixture invented (the existing field-selection fixture is reused), and no test asserts on internal structure beyond the error text it must carry.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run` subtests named "it does X", `t.Helper()` on every helper taking `*testing.T`, helpers living in the test file that owns them.
- SOLID principles: Good. `readmeCoverageErrors` is a pure function over (fences, samples, exemptions), which is what makes both negative cases testable without touching the README; `readmeFence` gained one field rather than a parallel structure.
- Complexity: Low. One pass over fences, one over exemptions; `fenceBody` keeps its single responsibility and now returns the prompt it already had to detect.
- Modern idioms: Yes — `slices.Sorted(maps.Keys(...))` for deterministic exemption ordering, named return values on `fenceBody` documenting the triple.
- Readability: Good. `firstLineOf` removes the three repeated `strings.Cut(body, "\n")` sites; `readmeAnchor`/`readmeClaim` name what was previously implicit in `findREADMEBlock`'s loop.
- Issues: None. The three new doc comments (:454-455, :471-472, :110-111) hold against the code and reference no task or phase.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "README content is unchanged; the diff touches `internal/cli/readme_samples_test.go` only, and `go test ./...` passes." — the first two clauses are settled by reading (commit b9782df7 touches the one file, README diff empty); the suite run is not. An executing pass over `go test ./internal/cli -run 'TestREADMESamplesMatchRenderedOutput|TestREADMEPromptedSamplesAreClaimed'` settles both the byte-for-byte rendering of the two new samples and the live coverage guard, which is the only part of this task reading cannot measure directly.
