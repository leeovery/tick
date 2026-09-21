TASK: free-text-round-trip-1-7 (tick-ca8507) — Corrections: extend the README's TOON explainer sentence so it names all three forms the tool emits

ACCEPTANCE CRITERIA:
- `README.md:421` names the tabular form, the named-field form and the inline list form, and every form it names appears in one of the two samples directly beneath it.
- Each shape present in the two samples — the `tasks[2]{…}:` header with rows, the bare `id:`/`title:`/`status:` head lines, the `tags[2]: auth,backend` and `refs[1]: …` inline lists — is covered by a form the sentence names.
- The "Designed for AI consumption" opening and the "Uses 30-60% fewer tokens than equivalent JSON" closing both survive the edit.
- `git diff README.md` shows changes on line 421 only; the fenced blocks at `423-427` and `429-451` are byte-identical to their pre-edit content.
- No file other than `README.md` is modified.

STATUS: complete

SPEC CONTEXT: §1 of the specification opens the work on the failure that a standard TOON reader dies on the first line of `tick show` output, because hand-assembled sections invented shapes the library does not produce. §5/§6 replace those with top-level named fields and library inline lists. §12.1 states the README is live documentation someone reads to learn the tool, so leaving it describing output the tool does not produce is shipping a defect. The explainer sentence is the prose counterpart of that rule: it described only the header-and-rows shape, which is exactly the parser a reader would write and exactly the parser that fails on the detail document.

IMPLEMENTATION:
- Status: Implemented
- Location: README.md:462 (the sentence; line 421 pre-edit — later phases added content above it); commit bef5f4cb, `README.md | 2 +-`, one insertion, one deletion, one file.
- Notes: The delivered sentence reads "Designed for AI consumption. A list of same-shaped rows becomes a tabular section — a `name[N]{cols}:` header declaring the schema once, followed by compact CSV-like rows; a single object becomes named `key: value` fields; a collection of scalars becomes an inline `name[N]: a,b` list. Uses 30-60% fewer tokens than equivalent JSON." All three named forms appear in the two samples directly beneath: the tabular form at README.md:465-468 (`tasks[2]{id,title,status,priority,type}:` plus two rows) and again inside the detail sample (`blocked_by[1]{id,title,status}:`, `children[0]{id,title,status}:`, `notes[1]{index,text,created}:`); the named-field form at README.md:472-478 (`id:`/`title:`/`status:`/`priority:`/`type:`/`created:`/`updated:`) and README.md:492 (`description:`); the inline list form at README.md:485 (`tags[2]: auth,backend`) and README.md:487 (`refs[1]: "https://github.com/org/repo/issues/42"`). Opening and closing clauses are byte-unchanged (visible in the commit diff). `git show bef5f4cb -- README.md` touches only the explainer line — both fenced blocks are outside the hunk and therefore byte-identical. `git show --stat` lists README.md alone. No stale copy of the old wording survives anywhere: grep for "Schema is declared once" and "CSV-like" across README.md, internal/cli/*.go and the specification returns only README.md:462. No other README prose contradicts the sentence — the remaining TOON prose (README.md:212, :321, :494) describes field selection, dep-tree edge lists and the C0 refusal, none of which restate the document shapes.
- Both samples beneath the sentence remain pinned to real rendered output, so the forms the prose names cannot silently drift from what the binary emits: internal/cli/readme_samples_test.go:340 renders `show tick-a1b2 --toon` against the block anchored on `id: tick-a1b2`, and :366 renders `list --toon` against the block anchored on `tasks[2]{id,title,status,priority,type}:`. Both anchors occur exactly once in README.md (lines 471 and 465), so the occurrence-1 claims in those samples resolve to these two blocks.

TESTS:
- Status: Adequate (prose-only change; the plan specifies no new tests, and none is warranted)
- Coverage: Behaviour is unchanged — no formatter, handler or test file is in the commit. The existing README suite is unaffected by construction: internal/cli/readme_samples_test.go reads only fenced blocks (`fencesIn` collects lines between ``` markers, :70-80) and never asserts on surrounding prose, so a prose-only edit cannot move TestREADMEToonSamplesDecode, TestREADMESamplesMatchRenderedOutput or TestREADMEPromptedSamplesAreClaimed. No test in internal/cli references the edited sentence's old or new text.
- Notes: None. Prose accuracy is not a property a unit test can assert without duplicating the sentence, and the samples it describes are already pinned against the binary.

CODE QUALITY:
- Project conventions: N/A (documentation prose; no Go code changed)
- SOLID principles: N/A
- Complexity: N/A
- Modern idioms: N/A
- Readability: Good — the three forms are named in the order a reader meets them in the samples below, each with the literal syntax it takes, and the sentence stays one sentence.
- Issues: None. The prose is true against the formatter: the detail document emits top-level named fields, and tags/refs emit as library inline lists, which is what the samples show and what the rendered-output tests pin.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./internal/cli -run TestREADMEToonSamplesDecode` stays green" and "`go test ./...` stays green" (the task's Tests section) — a suite run settles these. Reading establishes the necessary condition (the commit changes prose only, and no test reads README prose), but green-ness itself is only observable by executing the suite.
