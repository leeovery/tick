---
status: in-progress
created: 2026-02-28
cycle: 1
phase: Traceability Review
topic: cli-enhancements
---

# Review Tracking: cli-enhancements - Traceability

## Findings

### 1. Note data model task (tick-6cd164) has empty description

**Type**: Incomplete coverage
**Spec Reference**: Notes section -- Data model, Validation
**Plan Reference**: Phase 4 / cli-enhancements-4-5 (tick-6cd164)
**Change Type**: update-task

**Details**:
Task tick-6cd164 "Note data model with validation and JSONL serialization" has an empty description. The plan index lists edge cases (empty note text, note exactly 500 chars, 501-char note, whitespace-only text) but the task itself contains no Problem, Solution, Outcome, Do, Acceptance Criteria, Tests, or any other content. An implementer would have to go back to the specification. The spec defines: Note struct with Text string and Created time.Time, Notes []Note on Task with omitempty, empty note text errors, max 500 chars, adding/removing notes updates Updated timestamp.

**Current**:
Task tick-6cd164 has title "Note data model with validation and JSONL serialization" and empty description.

**Proposed**:
Description for tick-6cd164:

```
Problem: Tasks have no way to store timestamped context entries. No Note data type, no Notes field on Task, no validation.

Solution: Add Note struct (Text string, Created time.Time) and Notes []Note to Task struct with json:"notes,omitempty". Implement validation for note text (non-empty, max 500 chars). Ensure JSONL round-trip.

Outcome: Task struct carries validated Notes field that serializes to/from JSONL. Note validation functions available for CLI handlers.

Do:
- Create Note struct in internal/task/ with Text string and Created time.Time
- Add Notes []Note to Task struct with json:"notes,omitempty" tag
- Add Notes to taskJSON struct and wire through MarshalJSON/UnmarshalJSON
- Implement ValidateNoteText(text string) error -- non-empty after trim, max 500 chars
- Implement NormalizeNoteText(text string) string -- trim whitespace
- Tests in internal/task/note_test.go or task_test.go

Acceptance Criteria:
- Note struct has Text string and Created time.Time fields
- Notes []Note with json:"notes,omitempty" on Task struct
- Notes round-trip through JSON marshal/unmarshal
- Empty notes slice omitted from JSON (omitempty)
- ValidateNoteText("") returns error
- ValidateNoteText with whitespace-only returns error
- ValidateNoteText with 500 chars returns nil
- ValidateNoteText with 501 chars returns error
- Backward compatible: existing tasks without notes deserialize with nil Notes

Tests:
- "it marshals notes field when set"
- "it omits notes field when empty"
- "it unmarshals notes field from JSON"
- "it unmarshals task without notes field (backward compat)"
- "it rejects empty note text"
- "it rejects whitespace-only note text"
- "it accepts note text at 500 chars"
- "it rejects note text at 501 chars"
- "it preserves note Created timestamp through round-trip"

Edge Cases:
- Empty note text: rejected by validation
- Whitespace-only text: trimmed then rejected as empty
- 500-char boundary: exactly 500 accepted, 501 rejected
- Note with Created time: preserved through JSON round-trip

Spec Reference: .workflows/specification/cli-enhancements/specification.md -- Notes section
```

**Resolution**: Pending
**Notes**:

---

### 2. Notes SQLite table task (tick-17373b) has empty description

**Type**: Incomplete coverage
**Spec Reference**: Notes section -- Storage (SQLite: task_notes table)
**Plan Reference**: Phase 4 / cli-enhancements-4-6 (tick-17373b)
**Change Type**: update-task

**Details**:
Task tick-17373b "Notes table in SQLite schema and Cache.Rebuild" has an empty description. The spec defines: SQLite table task_notes(task_id, text, created) populated during Cache.Rebuild(). The plan index lists edge cases (empty notes slice, rebuild clearing stale notes, note ordering preserved) but no implementation detail exists in the task.

**Current**:
Task tick-17373b has title "Notes table in SQLite schema and Cache.Rebuild" and empty description.

**Proposed**:
Description for tick-17373b:

```
Problem: SQLite cache has no table for notes. Cannot query or display notes via cache.

Solution: Add task_notes(task_id, text, created) table to SQLite schema, extend Cache.Rebuild() to insert note rows per task. Follows task_tags and task_refs junction table pattern.

Outcome: After rebuild, every task's notes stored in task_notes. Table cleared and repopulated on each rebuild. Note ordering preserved by created timestamp.

Do:
- Add CREATE TABLE IF NOT EXISTS task_notes (task_id TEXT NOT NULL, text TEXT NOT NULL, created DATETIME NOT NULL) to schemaSQL
- Add CREATE INDEX IF NOT EXISTS idx_task_notes_task_id ON task_notes(task_id)
- In Cache.Rebuild(), add DELETE FROM task_notes alongside existing DELETEs
- Prepare INSERT INTO task_notes (task_id, text, created) VALUES (?, ?, ?)
- In task loop, iterate t.Notes and insert each note row with text and created timestamp
- Tests in internal/storage/cache_test.go

Acceptance Criteria:
- task_notes table with task_id, text, created columns
- Cache.Rebuild() inserts one row per note per task
- Cache.Rebuild() clears all task_notes before inserting
- Empty Notes slice -> zero rows
- Multiple notes -> rows ordered by created timestamp
- Rebuild twice produces same result (idempotent)
- Note ordering preserved through rebuild

Tests:
- "it creates task_notes table in schema"
- "it populates task_notes during rebuild for task with notes"
- "it inserts no rows for task with empty notes slice"
- "it clears stale notes on rebuild"
- "it preserves note ordering by created timestamp"
- "it handles rebuild with multiple tasks having different note sets"

Edge Cases:
- Empty notes slice: zero rows, no errors
- Stale notes on rebuild: DELETE before INSERT ensures clean state
- Note ordering: created timestamp preserves chronological order

Spec Reference: .workflows/specification/cli-enhancements/specification.md -- Notes section, Storage subsection
```

**Resolution**: Pending
**Notes**:

---

### 3. Note add subcommand task (tick-a4c883) has empty description

**Type**: Incomplete coverage
**Spec Reference**: Notes section -- Subcommands (tick note add)
**Plan Reference**: Phase 4 / cli-enhancements-4-7 (tick-a4c883)
**Change Type**: update-task

**Details**:
Task tick-a4c883 "Note add subcommand" has an empty description. The spec defines: `tick note add <id> <text>` appends a timestamped note, text is multi-word from remaining args after ID, adding a note updates Updated timestamp, empty text errors, max 500 chars. The plan index lists edge cases (missing id, missing text, text from multiple remaining args, task not found) but no task content exists.

**Current**:
Task tick-a4c883 has title "Note add subcommand" and empty description.

**Proposed**:
Description for tick-a4c883:

```
Problem: No CLI command exists to add notes to tasks. Users cannot append timestamped context entries.

Solution: Add tick note add <id> <text> subcommand. Parse remaining args after ID as note text (multi-word). Resolve ID via ResolveID, validate text, create Note with current timestamp, append to task's Notes, update task's Updated timestamp via store.Mutate.

Outcome: Users can run tick note add a3f "Started investigating auth flow" to append timestamped notes. Validation errors for missing/empty/too-long text.

Do:
- Add note command dispatch in app.go: route "note" to subcommand handler
- Implement RunNoteAdd in internal/cli/note.go: parse args, resolve ID, validate text, mutate
- Parse args: args[0] = task ID, remaining args joined as note text (strings.Join(args[1:], " "))
- Resolve ID via store.ResolveID after opening store
- Validate note text via task.ValidateNoteText (non-empty, max 500 chars)
- In store.Mutate callback: create Note{Text: text, Created: time.Now()}, append to task.Notes, set task.Updated = time.Now()
- Output confirmation message via formatter
- Tests in internal/cli/note_test.go

Acceptance Criteria:
- tick note add <id> "note text" appends note with current timestamp
- tick note add <id> multi word text joins remaining args as text
- tick note add without text errors
- tick note add with empty text (just whitespace) errors
- tick note add with text exceeding 500 chars errors
- tick note add with invalid/not-found ID errors appropriately
- Note appended to end of Notes slice
- Task's Updated timestamp set to current time after adding note
- Note visible in tick show <id> after adding

Tests:
- "it adds a note to a task"
- "it joins multiple remaining args as note text"
- "it errors when no text provided"
- "it errors when text is whitespace-only"
- "it errors when text exceeds 500 chars"
- "it errors when task ID not found"
- "it resolves partial ID for note add"
- "it updates task Updated timestamp after adding note"
- "it appends note to existing notes"

Edge Cases:
- Missing text: error when no args after ID
- Multi-word text: remaining args after ID joined with spaces
- Task not found: ResolveID error propagated
- Missing ID: error when no args at all

Spec Reference: .workflows/specification/cli-enhancements/specification.md -- Notes section, Subcommands subsection
```

**Resolution**: Pending
**Notes**:

---

### 4. Note remove subcommand task (tick-7402d4) has empty description

**Type**: Incomplete coverage
**Spec Reference**: Notes section -- Subcommands (tick note remove), Validation
**Plan Reference**: Phase 4 / cli-enhancements-4-8 (tick-7402d4)
**Change Type**: update-task

**Details**:
Task tick-7402d4 "Note remove subcommand" has an empty description. The spec defines: `tick note remove <id> <index>` removes by 1-based position, index must be >= 1 and <= number of existing notes, out-of-bounds errors, removing a note updates Updated timestamp. The plan index lists edge cases (index 0, index exceeding note count, negative index, non-integer index, task with no notes) but no task content exists.

**Current**:
Task tick-7402d4 has title "Note remove subcommand" and empty description.

**Proposed**:
Description for tick-7402d4:

```
Problem: No way to remove a note from a task. Users cannot correct mistakes or remove outdated context.

Solution: Add tick note remove <id> <index> subcommand. Parse ID and 1-based index from args, resolve ID via ResolveID, validate index bounds, remove note at position, update task's Updated timestamp via store.Mutate.

Outcome: Users can run tick note remove a3f 2 to remove the second note. Index validation catches all boundary errors.

Do:
- Add "remove" subcommand routing in note command dispatch
- Implement RunNoteRemove in internal/cli/note.go: parse args, resolve ID, parse index, validate, mutate
- Parse args: args[0] = task ID, args[1] = index (strconv.Atoi)
- Validate index: must be >= 1 and <= len(task.Notes); error if out of bounds
- In store.Mutate callback: remove note at index-1 (0-based slice), set task.Updated = time.Now()
- Handle task with no notes: index validation catches (any index > 0 exceeds len 0)
- Output confirmation message via formatter
- Tests in internal/cli/note_test.go

Acceptance Criteria:
- tick note remove <id> 1 removes the first note
- tick note remove <id> 0 errors (must be >= 1)
- tick note remove <id> 5 on task with 3 notes errors (exceeds count)
- tick note remove <id> -1 errors
- tick note remove <id> abc errors (non-integer)
- tick note remove on task with no notes errors for any index
- Task's Updated timestamp set to current time after removing note
- Remaining notes preserved in original order
- Note visible removed in tick show <id> after removing

Tests:
- "it removes a note by 1-based index"
- "it removes the last note from a task"
- "it errors on index 0"
- "it errors on index exceeding note count"
- "it errors on negative index"
- "it errors on non-integer index"
- "it errors when task has no notes"
- "it resolves partial ID for note remove"
- "it updates task Updated timestamp after removing note"
- "it preserves remaining notes order after removal"

Edge Cases:
- Index 0: error, must be >= 1
- Index > note count: out of bounds error
- Negative index: error
- Non-integer index: parse error
- Task with no notes: any valid index exceeds bounds

Spec Reference: .workflows/specification/cli-enhancements/specification.md -- Notes section, Subcommands and Validation subsections
```

**Resolution**: Pending
**Notes**:

---

### 5. Notes display task (tick-8b1edf) has empty description

**Type**: Incomplete coverage
**Spec Reference**: Notes section -- Display (in tick show)
**Plan Reference**: Phase 4 / cli-enhancements-4-9 (tick-8b1edf)
**Change Type**: update-task

**Details**:
Task tick-8b1edf "Notes display in show output and all formatters" has an empty description. The spec defines: notes shown chronologically (most recent last), display format with timestamp `YYYY-MM-DD HH:MM` and text. The plan index lists edge cases (no notes, multiple notes, long text) but no task content exists.

**Current**:
Task tick-8b1edf has title "Notes display in show output and all formatters" and empty description.

**Proposed**:
Description for tick-8b1edf:

```
Problem: Show command doesn't query or display notes. No formatter renders notes.

Solution: Extend queryShowData to query task_notes, add notes to showData and TaskDetail, update all three formatters' FormatTaskDetail. Show only, not list. Notes shown chronologically (most recent last).

Outcome: tick show <id> displays notes with timestamps. Format: YYYY-MM-DD HH:MM followed by text.

Do:
- format.go: Add Notes []NoteDetail (with Text string, Created time.Time) to TaskDetail
- show.go: Add notes to showData. Query SELECT text, created FROM task_notes WHERE task_id = ? ORDER BY created ASC. Copy to TaskDetail.
- PrettyFormatter: Add Notes: section. Each note on its own line indented: "  YYYY-MM-DD HH:MM  note text". Omit section when no notes.
- ToonFormatter: Add notes section with timestamp and text fields. Omit when empty.
- JSONFormatter: Add Notes to jsonTaskDetail with timestamp and text. Non-nil empty slice for [].
- Formatter tests.

Acceptance Criteria:
- tick show <id> displays notes chronologically (most recent last)
- Pretty format: "Notes:" header followed by indented timestamp+text lines
- Pretty timestamp format: YYYY-MM-DD HH:MM
- Toon format: notes section with structured fields
- JSON format: notes array with text and created fields, empty [] not null
- Omit notes section (pretty/toon) when no notes; JSON shows []
- Multiple notes all displayed in chronological order
- Long note text displayed without truncation

Tests:
- "it displays notes in pretty format show output"
- "it omits notes section in pretty format when task has no notes"
- "it displays notes chronologically (most recent last)"
- "it formats note timestamps as YYYY-MM-DD HH:MM"
- "it displays notes in toon format show output"
- "it displays notes in json format show output"
- "it shows empty notes array in json format when no notes"
- "it displays multiple notes in order"
- "it displays note with long text without truncation"

Edge Cases:
- No notes: pretty/toon omit section; JSON shows []
- Multiple notes: all displayed in chronological order (ORDER BY created ASC)
- Long note text: displayed fully, no truncation

Spec Reference: .workflows/specification/cli-enhancements/specification.md -- Notes section, Display subsection
```

**Resolution**: Pending
**Notes**:

---

### 6. Refs not filterable requirement not explicitly noted in plan

**Type**: Missing from plan
**Spec Reference**: External References section -- Filtering: "Not filterable on list, ready, blocked"
**Plan Reference**: Phase 4 acceptance criteria
**Change Type**: add-to-task

**Details**:
The spec explicitly states refs are "Not filterable on list, ready, blocked. Refs are a 'look up' thing -- visible on show, followed as links. Add later if demand emerges." The plan correctly omits any filtering task for refs, but this deliberate omission is not documented anywhere in the plan's Phase 4 acceptance criteria or task content. While the absence of a filtering task is correct behavior, the spec's explicit decision should be acknowledged so an implementer doesn't wonder whether filtering was accidentally forgotten.

However, on further analysis, the plan's Phase 4 acceptance criteria and the refs display task (tick-4b4e4b) both focus exclusively on show output. The spec decision is implicitly honored by the absence of any filtering task or flag mention. This is adequate -- the plan need not document every feature it does NOT implement. Withdrawing this finding.

**Resolution**: Withdrawn
**Notes**: The plan correctly omits refs filtering. The absence is sufficient -- plans don't need to enumerate features they deliberately exclude.

---

### 7. Tag filter input normalization not explicit in filtering task

**Type**: Incomplete coverage
**Spec Reference**: Tags section -- Filtering: "Filter input normalized (trimmed, lowercased) before matching"
**Plan Reference**: Phase 3 / cli-enhancements-3-5 (tick-56001c)
**Change Type**: update-task

**Details**:
The spec explicitly states "Filter input normalized (trimmed, lowercased) before matching" for tag filtering. The task tick-56001c mentions "normalize via task.NormalizeTag" in the Do section and has "normalize first, then validate" in Edge Cases, plus a test "it normalizes tag filter input". However, the Acceptance Criteria do not include a criterion for input normalization of filter values (e.g., `--tag UI` matching tasks tagged `ui`). Adding an explicit acceptance criterion ensures this spec requirement is verifiably tested.

**Current**:
In tick-56001c acceptance criteria:
```
- tick list --tag ui returns only tasks tagged ui
- tick list --tag ui,backend returns tasks with both tags (AND)
- tick list --tag ui,backend --tag api returns (ui AND backend) OR api
- tick ready --tag ui and tick blocked --tag ui filter correctly
- Composes with --status, --priority, --parent, --count
- Invalid kebab-case in filter errors
- No matching tasks returns empty list
- Single --tag value works as single-element AND group
```

**Proposed**:
In tick-56001c acceptance criteria (add after "tick ready --tag ui and tick blocked --tag ui filter correctly"):
```
- tick list --tag ui returns only tasks tagged ui
- tick list --tag ui,backend returns tasks with both tags (AND)
- tick list --tag ui,backend --tag api returns (ui AND backend) OR api
- tick ready --tag ui and tick blocked --tag ui filter correctly
- --tag UI normalizes to ui before matching (filter input trimmed and lowercased)
- Composes with --status, --priority, --parent, --count
- Invalid kebab-case in filter errors
- No matching tasks returns empty list
- Single --tag value works as single-element AND group
```

**Resolution**: Pending
**Notes**:

---

### 8. Type filter input normalization acceptance criterion partially covered

**Type**: Incomplete coverage
**Spec Reference**: Task Types section -- Filtering: "Filter input normalized (trimmed, lowercased) before matching"
**Plan Reference**: Phase 2 / cli-enhancements-2-4 (tick-3357ef)
**Change Type**: update-task

**Details**:
The spec states "Filter input normalized (trimmed, lowercased) before matching" for type filtering. Task tick-3357ef does include "tick list --type BUG normalizes to bug and filters correctly" in acceptance criteria, which covers case normalization. However, it lacks trimming coverage. The spec says "trimmed" explicitly. Adding an acceptance criterion for trimmed input.

Actually, on review, the ValidateType/NormalizeType functions in task 2-1 already handle trimming. The filter task uses NormalizeType before ValidateType. The spec says "trimmed, lowercased" and the task covers lowercase via "BUG normalizes to bug". Trimming is implementation detail handled by NormalizeType. The acceptance criterion adequately covers the spec intent -- `--type BUG` tests normalization, and the Do section explicitly calls task.NormalizeType which trims. This is sufficient.

**Resolution**: Withdrawn
**Notes**: NormalizeType (from task 2-1) handles trimming. The filter task correctly delegates to it. Coverage adequate.

---

### 9. Refs input trimming before validation not explicit in task acceptance criteria

**Type**: Incomplete coverage
**Spec Reference**: External References section -- Validation: "Input trimmed before validation"
**Plan Reference**: Phase 4 / cli-enhancements-4-1 (tick-e7bb22)
**Change Type**: update-task

**Details**:
The spec states "Input trimmed before validation" for refs. Task tick-e7bb22 includes "Input trimmed before validation" in acceptance criteria, so this is actually covered. Withdrawing.

**Resolution**: Withdrawn
**Notes**: Already covered in acceptance criteria.

---

### 10. Notes "no tick note list" decision not documented

**Type**: Missing from plan
**Spec Reference**: Notes section -- Subcommands: "No tick note list -- view notes via tick show. Can add later if needed."
**Plan Reference**: Phase 4
**Change Type**: add-to-task

**Details**:
The spec explicitly states there is no `tick note list` subcommand -- users view notes via `tick show`. The plan correctly has no task for this, but the note add/remove tasks don't mention that `list` is intentionally excluded. Similar to finding #6, the absence is sufficient -- plans don't enumerate excluded features.

**Resolution**: Withdrawn
**Notes**: Absence of a `tick note list` task is sufficient. No need to document excluded features.

---

### 11. Refs display not in list output requirement

**Type**: Missing from plan
**Spec Reference**: External References section -- Display: "List output: not shown"
**Plan Reference**: Phase 4 / cli-enhancements-4-4 (tick-4b4e4b)
**Change Type**: update-task

**Details**:
The spec states refs should not be shown in list output. Task tick-4b4e4b says "Show only, not list" in the Do section, which covers this. Withdrawing.

**Resolution**: Withdrawn
**Notes**: Already covered in Do section.

---

### 12. Tags "not displayed in list output" not in tags display task acceptance criteria

**Type**: Incomplete coverage
**Spec Reference**: Tags section -- Display: "List output: not shown (variable-length, would clutter the table)"
**Plan Reference**: Phase 3 / cli-enhancements-3-3 (tick-d17558)
**Change Type**: update-task

**Details**:
The spec explicitly states tags should not be shown in list output. Task tick-d17558 says "FormatTaskList unchanged" in the Do section, and the plan index acceptance says "Tags displayed in show output; not displayed in list output." However, the task's own acceptance criteria do not include a negative criterion confirming tags are absent from list output. Adding one ensures an implementer doesn't accidentally add tags to list format.

**Current**:
In tick-d17558 acceptance criteria:
```
- tick show <id> displays tags for task with tags
- Omits tags section (pretty/toon) or shows empty array (JSON) for no tags
- Pretty: Tags:     tag1, tag2, tag3 (comma-space separated)
- Toon: tags field in task section
- JSON: "tags" always present as array (empty [], never null)
- Tags sorted alphabetically (ORDER BY tag)
- 10 tags all displayed
```

**Proposed**:
In tick-d17558 acceptance criteria (add after "10 tags all displayed"):
```
- tick show <id> displays tags for task with tags
- Omits tags section (pretty/toon) or shows empty array (JSON) for no tags
- Pretty: Tags:     tag1, tag2, tag3 (comma-space separated)
- Toon: tags field in task section
- JSON: "tags" always present as array (empty [], never null)
- Tags sorted alphabetically (ORDER BY tag)
- 10 tags all displayed
- Tags not shown in list output (show/detail only)
```

**Resolution**: Pending
**Notes**:

---

### 13. Notes "adding or removing updates Updated timestamp" only in plan acceptance but missing from note add/remove task details

**Type**: Incomplete coverage
**Spec Reference**: Notes section -- Data model: "Adding or removing a note updates the task's Updated timestamp"
**Plan Reference**: Phase 4 / cli-enhancements-4-7 (tick-a4c883), cli-enhancements-4-8 (tick-7402d4)
**Change Type**: update-task

**Details**:
This is already covered in findings #3 and #4, which propose full task descriptions including the Updated timestamp requirement. The proposed content for both tick-a4c883 and tick-7402d4 includes acceptance criteria and Do steps for updating the task's Updated timestamp. No separate finding needed.

**Resolution**: Withdrawn
**Notes**: Covered by findings #3 and #4.
