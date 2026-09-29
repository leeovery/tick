TASK: Shared Line Reader With No Length Ceiling (oversized-text-field-bricks-the-store-1-1, tick-1150de)

ACCEPTANCE CRITERIA:
- A store whose tasks.jsonl holds a task line of exactly 65,536 bytes (newline excluded): `tick list` succeeds and lists that task (§1.1, §2.1)
- Task lines of exactly 65,535 bytes, exactly 65,536 bytes and about 1 MiB, newline excluded: the shared reader returns each line whole, and a store holding all three lists all three tasks (§2.1, §8.1)
- A store whose lines end in CRLF: every task reads exactly as it does from the same store with LF endings (§2.2)
- A store whose final task line has no trailing newline: that task is read. When that line ends in one `\r`, the `\r` is stripped and the task still reads (§2.2, Corrigendum 2026-09-28)
- An otherwise valid store whose file ends `…}\n\r`: `tick list` succeeds, because the final `\r` is an empty, skipped line (§2.2, Corrigendum 2026-09-28)
- A store with empty lines between its tasks: every task is read and the empty lines are skipped (§2.2)
- A store whose line 2 is empty and whose line 3 holds only whitespace: `tick list` fails, naming line 3 (§2.2)
- The reader imposes no per-line size limit, neither Go's default scanner limit nor any larger one set in its place (§2.1)

STATUS: complete

SPEC CONTEXT: §1.2 traces the defect to ParseJSONL's bufio.Scanner at Go's default 64 KiB token size. §2.1 requires every reader of tasks.jsonl to accept a line of any length, with no replacement limit. §2.2 requires one shared line reader carrying the store's existing line definition exactly: split at "\n", drop one "\r" before it (and one at the end of an unterminated final line, per the 2026-09-28 corrigendum), skip lines that are empty once the terminator is removed, keep whitespace-only lines as ordinary lines, and number from 1 with skipped lines counted. §8.1 names the 65,535 / 65,536 / ~1 MiB regression guard. §8.6 names TestParseJSONL and TestReadJSONL for revisiting. Doctor's move onto the reader belongs to Phase 2.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/jsonl/lines.go:29-48 — `Lines(r io.Reader) iter.Seq2[Line, error]` over `bufio.Reader.ReadBytes('\n')`, which has no length limit and returns a freshly allocated slice. The accurate "owned by the caller" doc at :18 depends on that allocation.
  - internal/jsonl/lines.go:50-53 — `trimTerminator` removes one "\n", then one "\r". This matches `bufio.ScanLines` and `dropCR` both for terminated lines and for the unterminated final line.
  - internal/jsonl/lines.go:38 — `len(text) > 0` gives the old `line == ""` skip. A whitespace-only line is still yielded.
  - internal/jsonl/lines.go:32 — `num` goes up on every read, skipped lines included. Trailing EOF reads with no content are never yielded, so they cannot move any reported number.
  - internal/storage/jsonl.go:95 — ParseJSONL ranges over `jsonl.Lines(bytes.NewReader(data))`. `bufio` is no longer imported in the storage package's production code.
  - internal/storage/store.go:164, :236, :365 — ReadTasks, Rebuild and readAndEnsureFresh all parse through ParseJSONL, so all three moved with it.
- Notes: I checked the old scanner loop (4e98f8fc^) against the new reader case by case: "a\n" (no phantom line), "a\n\r" (empty final line skipped), "\r\r\n" (keeps one "\r", non-empty, fails to load as before), "a\rb\n" (inner "\r" kept), an unterminated final line with and without a trailing "\r", and empty input. Every case behaves the same, so stores the old reader could read are read identically. As delivered by 4e98f8fc, the task kept the old `failed to parse line N:` detail, as instructed. The current `line N (id):` form comes from Task 1-6 and is outside this task. The only production `bufio` line readers left in internal/ are the beads importer (external file, Task 1-5, §2.3) and remove.go's stdin confirmation. Neither reads tasks.jsonl.

TESTS:
- Status: Adequate
- Coverage:
  - Reader (internal/jsonl/lines_test.go): 65,535, 65,536 and 1 MiB lines returned whole (:58). A 1 MiB unterminated final line (:65). CRLF (:70). Only one "\r" stripped (:74). An inner "\r" kept (:78). Unterminated final line (:82) and the same with a trailing "\r" (:86). A final lone "\r" skipped (:90). Empty-line skipping with numbering (:95, including a "\r\n" empty line). A whitespace-only line returned (:99). Empty input (:102). A read error names its line and wraps the cause (:106). Consumer break (:120).
  - Parser (internal/storage/jsonl_test.go): TestParseJSONL now covers the three sizes, including exact description round-trip (:572), CRLF against LF (:598), the final-line cases (:616, :621, :626) and the whitespace-only line 3 after an empty line 2 (:631). TestReadJSONL gains an over-64 KiB file case (:208).
  - CLI (internal/cli/store_line_reading_test.go): one test per acceptance criterion through `tick list`: exactly 65,536 (:50), all three sizes (:55), CRLF output identical to LF (:64), the final-line cases (:74, :79), `…}\n\r` (:84), empty lines (:89), and the whitespace-only line 3 with exact stderr (:94).
- Notes: The sized fixtures (`sizedTaskLine`, `taskLineOfSize`) compute padding so each line is exactly the stated size with the newline excluded. The 65,536 case therefore sits exactly at the size the old scanner refused, and 65,535 at the largest it accepted. JSON accepts a trailing "\r" as whitespace, so the store-level and CLI-level CRLF and final-"\r" tests would still pass if stripping broke. The exact-text reader assertions (lines_test.go:70, :86, :90, :95) are what catch a stripping regression, so the behaviour is still guarded. The parser and CLI layers repeat several cases, but each layer checks something the acceptance criteria name at that layer ("the shared reader returns…", "`tick list` succeeds…").

CODE QUALITY:
- Project conventions: Followed. Stdlib `testing` only, `t.Run` subtests named "it …", `t.Helper()` on helpers, `t.TempDir()` isolation, errors wrapped with `%w`, and the modern `iter.Seq2` range-over-func.
- SOLID principles: Good. `jsonl` has one job (framing lines) and no dependencies on other internal packages, so doctor can reach it without importing storage.
- Complexity: Low
- Modern idioms: Yes (`iter.Seq2`, `errors.Is`, range-over-func)
- Readability: Good. The doc comments on `Line` and `Lines` match the code: skip rule, "\r" handling, numbering, the single final error pair and caller-owned `Text`.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
