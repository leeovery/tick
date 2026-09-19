## Attempt 1

ISSUES:
- `internal/cli/toon_formatter_test.go:1428-1434` — the subtest "it leaves unfiltered output unchanged" asserts nothing. `richDetail()` never sets `Fields`, so `selected := detail; selected.Fields = nil` is a no-op copy and line 1432 compares `f.FormatTaskDetail(x)` with `f.FormatTaskDetail(x)` — a deterministic function against identical input. The guard cannot fail for any implementation of the nil path. The task's test list specified this test as "a nil selection renders the string the existing full-document assertions expect"; what shipped pins only the key set (line 1436), leaving section order, blank lines and values in the nil path unasserted by the test that claims to cover them.
  FIX: replace lines 1429-1434 with a byte-exact expectation. Build the document once (`result := f.FormatTaskDetail(richDetail())`), compare it to a `want` string holding the full 15-key document for `richDetail()` (head block, `blocked_by`, `children`, `tags`, `refs`, `notes`, `description`, joined by `"\n\n"`), then keep the existing `decodeToonDoc` + `assertToonKeySet` lines against `result`. This is the assertion the task asked for and it fails on any drift in the nil path.
  ALTERNATIVE: delete lines 1429-1434 and rename the subtest to "it renders every key when no selection is given", relying on the pre-existing full-document tests (`TestToonFormatter` at `toon_formatter_test.go:65`, `TestToonTaskDetailConformance` at `toon_decode_test.go:160`) for byte-level coverage. Cheaper and honest, but the phase never gets a single byte-exact pin of the filtered formatter's unfiltered output. The reviewer recommends the exact-string version.
  CONFIDENCE: high

COMMENT_CORRECTIONS:
- `internal/cli/toon_formatter.go:68` — the change made "full details" conditional: with `detail.Fields` set the method renders a subset, and the doc comment does not say the field exists.
  OLD: // FormatTaskDetail renders a single task with full details in multi-section TOON format.
  NEW: // FormatTaskDetail renders a single task in multi-section TOON format, narrowed to detail.Fields when it is set.

NOTES:
- `assertToonKeyOrder` (`toon_formatter_test.go:1316`) orders by `strings.Index` in two subtests whose output is fully deterministic (`--field status,title` renders exactly `title: Add retry to the sync worker\nstatus: in_progress`). code-quality.md lists substring assertions on deterministic output as an anti-pattern; exact comparison would pin content as well as order. Non-blocking — the guard does discriminate on order today.
- `RunShow`'s updated doc comment ("narrowed to the selected fields when one was given") is true for toon and not yet for `--pretty`/`--json`, which ignore `Fields` until tasks 4-4/4-5. Deliberately not raised as a correction: the sentence delegates narrowing to the Formatter, and correcting it now guarantees a second correction two tasks from now.
- `.tick/tasks.jsonl` is modified in the working tree (dogfooding tracker), not reviewed as code.
