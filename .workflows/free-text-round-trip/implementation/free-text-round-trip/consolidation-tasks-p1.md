# Consolidation Tasks: Free Text Round Trip (Phase 1)

## Task 1: Corrections
placement: phase 1
severity: corrections

**Problem**: The README's TOON explainer sentence describes only one of the two shapes the tool now emits. `README.md:421` reads "Designed for AI consumption. Schema is declared once in the header; rows are compact CSV-like lines." Directly beneath it, the `tick show` sample this phase rewrote (`README.md:429-451`) opens with seven bare `key: value` lines, carries two inline `name[N]: a,b` lists, and ends with a quoted scalar description. None of those forms is named anywhere in the README. A contributor or agent writing a reader from the prose writes a header-and-rows parser, and it fails on the first line of every task-detail document — the exact failure §1 of the specification exists to remove, reintroduced in the document people read to learn the tool. Four tasks jointly produced the new shape (1-1 head, 1-2 tags/refs, 1-3 description, 1-4 notes index); task 1-6 corrected the samples but not the prose above them, and no later phase touches that sentence — the README tasks in phases 2 through 5 replace samples and add flag documentation only.

**Solution**: One edit.

- `README.md:421` — extend the explainer sentence to name both forms the tool emits: a tabular section (`name[N]{cols}:` plus CSV-like rows) for a list of same-shaped rows, named `key: value` fields for a single object, and an inline `name[N]: a,b` list for a collection of scalars. Both samples beneath the sentence are then covered by the text above them. The samples themselves stay exactly as they are — they were verified byte-accurate against the built binary during the sweep.

**Outcome**: A reader who learns tick's machine format from the README's prose alone writes a parser that handles every document the tool emits, not only the task-list table.

**Do**:
- Rewrite the middle sentence of `README.md:421` — currently "Schema is declared once in the header; rows are compact CSV-like lines." — so it names all three forms the tool emits: a tabular section (`name[N]{cols}:` header plus CSV-like rows) for a list of same-shaped rows, named `key: value` fields for a single object, and an inline `name[N]: a,b` list for a collection of scalars.
- Keep the sentence's opening ("Designed for AI consumption.") and closing ("Uses 30-60% fewer tokens than equivalent JSON.") as they stand; only the middle clause changes.
- Leave both fenced samples beneath it byte-unchanged — the list sample at `README.md:423-427` and the `tick show` sample at `README.md:429-451`. Task 1-6 verified them against the built binary; the prose is what is wrong, not the output.
- Change nothing outside `README.md`: no formatter code, no tests.

**Acceptance Criteria**:
- [ ] `README.md:421` names the tabular form, the named-field form and the inline list form, and every form it names appears in one of the two samples directly beneath it.
- [ ] Each shape present in the two samples — the `tasks[2]{…}:` header with rows, the bare `id:`/`title:`/`status:` head lines, the `tags[2]: auth,backend` and `refs[1]: …` inline lists — is covered by a form the sentence names.
- [ ] The "Designed for AI consumption" opening and the "Uses 30-60% fewer tokens than equivalent JSON" closing both survive the edit.
- [ ] `git diff README.md` shows changes on line 421 only; the fenced blocks at `423-427` and `429-451` are byte-identical to their pre-edit content.
- [ ] No file other than `README.md` is modified.

**Tests**: Prose-only correction — behaviour is unchanged, no new tests, and no existing test's semantics are touched.
- `go test ./internal/cli -run TestREADMEToonSamplesDecode` stays green: it reads `README.md`, anchors on `tasks[3]{id,title,status,priority,type}:`, `tasks[2]{id,title,status,priority,type}:` and `id: tick-a1b2`, and decodes each block with the TOON library — leaving the samples untouched keeps all four subtests passing.
- `go test ./...` stays green.
