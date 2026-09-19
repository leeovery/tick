# Consolidation Findings: free-text-round-trip (Phase 1)

## Findings

### F1: README's TOON explainer describes only the tabular shape the detail document no longer opens with

- **Class**: drift
- **Failure**: A reader learning tick's agent format from README `### TOON` is told "Schema is declared once in the header; rows are compact CSV-like lines", and is then handed a `tick show` sample that opens with seven bare `key: value` lines, carries two inline `name[N]: a,b` lists and ends with a quoted scalar. Nothing in the README says those forms exist. A contributor or agent that writes a reader from the prose writes a header-and-rows parser and it fails on the first line of every task-detail document — the exact failure the specification's §1 exists to remove, reintroduced in the document people read to learn the tool. Noticed the first time someone parses `tick show` output from the README's description rather than from a TOON library, and by any reader who cannot tell from the README whether the named-field head is intentional or a typo.
- **Evidence**:
  - `README.md:421` — the explainer sentence, unchanged by this phase and now covering only one of the two shapes the tool emits.
  - `README.md:423-427` — the tabular sample the sentence does describe.
  - `README.md:429-451` — the detail sample this phase rewrote: named-field head (`430-436`), inline lists (`443`, `445`), quoted description (`450`). Verified byte-accurate against the built binary; the samples are right, the prose above them is not.
  - `internal/cli/toon_formatter.go:80-111` — `FormatTaskDetail` now emits both shapes in one document (`buildTaskSection` → `encodeToonFields` at `:84`, `encodeToonSection` for tags/refs at `:94`/`:99`, `encodeToonFields` for description at `:107`).
  - Caused by four tasks jointly: `eb88e700` (head), `fd989b94` (tags/refs), `cfcc96bb` (description), `3376f7a9` (notes index); `a29d0ca1` corrected the samples but not the sentence above them. No task owned the prose, and no later phase does either — the README tasks in phases 2, 3, 4 and 5 replace samples and add flag documentation only (`phase-2-tasks.md` Task 6, `phase-3-tasks.md` Task 6, `phase-4-tasks.md` Task 8, `phase-5-tasks.md` Task 6).
- **Proposed shape**: Extend `README.md:421` to name both forms the tool emits — a tabular section (`name[N]{cols}:` plus CSV-like rows) for a list of same-shaped rows, named `key: value` fields for a single object, and an inline `name[N]: a,b` list for a collection of scalars — so both samples beneath it are covered by the text above them. Samples stay as they are; this is one sentence, in the section both samples already sit in.

## Comment Corrections

Eight restatement comments in `internal/cli/toon_formatter.go`. The seven numbered ones narrate the `append` calls they sit above and re-state the guard on the very next line; the phase rewrote the first of them and rewrote the code under four more, leaving a numbered commentary on a function whose sections are self-evident from their builder names. Each carries nothing the code cannot, so each is deleted rather than reworded.

- `internal/cli/toon_formatter.go:83` — restates `buildTaskSection(detail.Task)`, and labels as "Section 1" the one piece of the document that is not a section
  OLD: 	// Section 1: the task's own fields as top-level named fields
  NEW:
- `internal/cli/toon_formatter.go:86` — restates `buildRelatedSection("blocked_by", …)` and its own count-zero fallback
  OLD: 	// Section 2: blocked_by (always present, even with count 0)
  NEW:
- `internal/cli/toon_formatter.go:89` — restates `buildRelatedSection("children", …)` and its own count-zero fallback
  OLD: 	// Section 3: children (always present, even with count 0)
  NEW:
- `internal/cli/toon_formatter.go:92` — restates the `if len(detail.Tags) > 0` guard on the next line
  OLD: 	// Section 4: tags (omitted when empty)
  NEW:
- `internal/cli/toon_formatter.go:97` — restates the `if len(detail.Refs) > 0` guard on the next line
  OLD: 	// Section 5: refs (omitted when empty)
  NEW:
- `internal/cli/toon_formatter.go:102` — restates `buildNotesSection(detail.Notes)` and its own count-zero fallback
  OLD: 	// Section 6: notes (always present, even with count 0)
  NEW:
- `internal/cli/toon_formatter.go:105` — restates the `if detail.Task.Description != ""` guard on the next line
  OLD: 	// Section 7: description (omitted when empty)
  NEW:
- `internal/cli/toon_formatter.go:344` — restates the `toon.NewObject(toon.Field{Key: name, Value: rows})` call directly beneath it; the phase rewrote the doc comment above this function and left the line
  OLD: 	// Build an Object with the named array field
  NEW:
