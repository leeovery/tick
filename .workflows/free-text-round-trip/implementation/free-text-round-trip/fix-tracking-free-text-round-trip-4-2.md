## Attempt 1

ISSUES:
- `internal/cli/show_fields.go:32-41` — the ten scalar/description entries use positional composite literals (`{showFieldScalar, func(d TaskDetail) string { ... }}`) while the five list entries on lines 42-46 use keyed form (`{kind: showFieldList}`). `.claude/skills/golang-code-style` states "Composite literals MUST use field names — positional fields break when the type adds or reorders fields", and these are the only positional struct literals in non-test production code in the repo; the comparable registry at `internal/task/state_machine.go:21-25` uses keyed fields throughout. Tasks 4-3 and 4-6 both extend this registry, so the mixed form is what the next author copies.
  FIX: Key both fields on the ten positional entries, e.g. `"id": {kind: showFieldScalar, bare: func(d TaskDetail) string { return d.Task.ID }},` and the same for `title`, `status`, `priority`, `type`, `parent`, `created`, `updated`, `closed`, `description`. Leave lines 42-46 as they are. No test change needed; re-run `gofmt -w ./internal` for alignment.
  CONFIDENCE: high

COMMENT_CORRECTIONS:
- `internal/cli/show.go:34-35` — the doc now claims an output path the function no longer always takes: a single-field selection returns without reaching the Formatter at all.
  OLD: // RunShow executes the show command: queries a single task by ID from SQLite and
// outputs its full details via the Formatter, including blocked_by, children, and description sections.
  NEW: // RunShow executes the show command: queries a single task by ID from SQLite and
// outputs the bare value of a single selected field, or the full detail document
// via the Formatter.

NOTES:
- `showField.kind == showFieldList` and `bare == nil` now redundantly encode the same fact. An entry that set both a list kind and a renderer would be self-contradictory with nothing to catch it. Risk is low (one file, 15 entries, both forms visible together) and the single-registry benefit outweighs it — no change asked for.
- `it ignores --pretty for a bare value` (`list_show_test.go:807`) compares the `--pretty` run against the unflagged run, but `runShow` sets `IsTTY: true`, so the unflagged run already resolves to pretty — that subtest is a tautology on its own. The criterion is still carried by the `--json` and `--toon` comparisons plus the exact-byte assertions, so nothing is untested; flagged only because the subtest's name promises more than it checks.
- `detail := showDataToTaskDetail(data)` now also runs on the quiet path, which discards it. Two `time.Parse` calls — negligible, and hoisting it keeps the bare branch where the task specified.
- `.tick/tasks.jsonl` is modified in the working tree (task status open → in_progress). Workflow bookkeeping, not part of the change set.
