TASK: free-text-round-trip-10-5 (tick-b57be3) — Corrections: README.md:494's TOON refusal paragraph states the pre-Tfree-text-round-trip-9-2 diagnostic rule ("— and the task, where the document covers one —"), telling a reader the task ID will not appear in a `tick list`/`ready`/`blocked` refusal when the shipped binary names it. Replace the restriction clause so the paragraph carries §3.1's rule whole.

ACCEPTANCE CRITERIA:
- [x] `grep -n 'where the document covers one' README.md` returns no line.
- [x] The paragraph names both halves of the rule — the field or section, and the task carrying the refused value — calls out the task-list row, and bounds the section-name-only case to a row that cannot be attributed.
- [x] The paragraph's first and last sentences are unchanged.
- [x] The whole diff is `README.md`, one paragraph: `git diff --stat` names no other file.
- [x] The sentence is true of the binary as it stands, independent of Task 2: `tick --toon list` over a project where one task's title carries `\x1b` names the `tasks` section and that task's ID, as `internal/cli/toon_refusal_test.go:163` asserts.
- [ ] `go test ./...` passes with no test file changed. — half settled by reading (no test file changed); the suite run is recorded under UNSETTLED.

STATUS: complete

SPEC CONTEXT:
§3.1 ("Output that must parse", specification.md:60-64) sets the refusal rule: "The command exits non-zero with a diagnostic naming the field or section it could not encode and the task that carries the refused value, and writes nothing to stdout. A task list names the offending row, not merely the section … Where the offending row cannot be identified, the section name alone stands." The corrigendum at specification.md:565 records why: the earlier clause scoped task-naming to single-task documents and left `tick list`/`ready`/`blocked` failing with `cannot encode section tasks as TOON: …` and no row, pushing an agent to bisection or to reading `.tick/tasks.jsonl` — §1's failure reached from the other side. §12.1 (specification.md:502-512) holds the README to what the tool produces: "It is live documentation someone reads to learn the tool … so leaving it describing output the tool does not produce is shipping a defect."

IMPLEMENTATION:
- Status: Implemented
- Location: README.md:494 (commit 3e5b5bf5, `README.md | 2 +-`, 1 insertion / 1 deletion, no other file)
- Notes:
  - The shipped sentence reads: "A command whose TOON document would carry such a value fails, naming the field or section it could not encode and the task carrying the refused value — the offending row, where the document is a task list — rather than printing a document without it; the section name stands alone only where no task can be attributed." Both halves of §3.1's rule are present, the task-list row is called out, and the section-name-only case is bounded to a row that cannot be attributed.
  - `grep -n 'where the document covers one' README.md` returns nothing (exit 1); the string survives nowhere else in the README. The only other README line mentioning a refusal is :212 (`--quiet` with a field selection), which is unrelated and untouched.
  - The paragraph's first sentence (C0 control characters / ANSI escapes) and last sentence (`--json` returns it, and so does `tick show <id> --field <name>`) are byte-identical in the diff.
  - Truth against the binary, verified by reading the code rather than the plan's wording: `internal/cli/toon_formatter.go:67` encodes the list's `tasks` section through `encodeToonSectionIdentified` with `func(r toonTaskRow) string { return r.ID }`; `sectionRefusal` (`toon_formatter.go:357-368`) re-encodes each row alone and records the first refused row's ID; `(*toonEncodeError).Error` (`toon_formatter.go:381-384`) renders `cannot encode section tasks of task <id> as TOON: …`. So a `tick --toon list` refusal names both the section and the row.
  - The trailing clause ("the section name stands alone only where no task can be attributed") holds across the other multi-task documents: the `changed` section is identified by ID (`toon_formatter.go:155`), and the only unidentified multi-task section is the dep-tree edge list (`toon_formatter.go:222`), whose `toonEdgeRow` carries nothing but `from`/`to` task IDs (`toon_formatter.go:164-168`) — values the encoder cannot refuse. Single-task documents get the ID attached downstream by `refusalForTask` (`toon_formatter.go:390-396`) via `toonDoc.join` (`:417-422`).
  - No drift: the task forbade touching any fenced sample, other README section, source, test, or `.workflows/` file, and the commit touches only README.md. The working tree carries no further README modification (`git status --porcelain` shows only `.workflows/` entries).

TESTS:
- Status: Adequate (no new test expected)
- Coverage: A prose-only correction; the plan explicitly expects no new test and no change to any test's semantics. The behaviour the paragraph now describes is already pinned by `internal/cli/toon_refusal_test.go:163` (`assertRefused(t, stdout, stderr, exitCode, "tasks", refused)` for `tick --toon list`) with `:164` asserting the encodable task stays unnamed, plus the `ready`/`blocked` table-driven cases at `:179-190` and the single-task field cases at `:153` and `:198`.
- Notes: No test reads the prose at README.md:494. The README suite's prose-scanning test is `TestREADMEDocumentsEndOfFlagsMarker` (`internal/cli/readme_samples_test.go:630-654`), which scans the Global Flags section, and `TestREADMEDocumentsFieldSelection` (`:563`) scans the `show` section via `readmeShowSection`. The remaining README tests — `TestREADMEToonSamplesDecode` (`:164`), `TestREADMESamplesMatchRenderedOutput` (`:425`), `TestREADMEPromptedSamplesAreClaimed` (`:514`) — operate on fenced samples, none of which the commit touched.

CODE QUALITY:
- Project conventions: N/A (documentation prose; no Go code changed)
- SOLID principles: N/A
- Complexity: N/A
- Modern idioms: N/A
- Readability: Good. The sentence carries two em-dash asides and a semicolon clause, which is dense, but it is grammatical, matches the surrounding README register, and each clause maps onto a distinct half of §3.1's rule.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...` passes with no test file changed." — the "no test file changed" half is settled: `git show --stat 3e5b5bf5` names only `README.md`. The suite run itself needs executing; reading finds no test that reads this prose, so no failure is expected.
