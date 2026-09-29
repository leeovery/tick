TASK: Doctor Reads Tasks.jsonl Through The Shared Line Reader (oversized-text-field-bricks-the-store-2-1, tick-552ab7)

ACCEPTANCE CRITERIA:
- An otherwise valid store with a fresh cache and a task line over 64 KiB mid-file, where tasks before the long line name tasks after it as parent or blocker: `tick doctor` exits 0 and reports `No issues found`, with no orphaned-reference error for the tasks after the long line.
- A store whose line 2 is empty and whose line 3 holds only whitespace: `tick doctor` exits 1 with a `JSONL syntax` failure naming line 3, the line `tick list` names in its own failure. The empty line is skipped but still counted.
- A fixture carrying CRLF-terminated lines, empty lines and a whitespace-only line among valid tasks: doctor and the store read the same lines and report the same line numbers. Doctor's only JSONL failure names the line that `tick list` names.
- A valid store whose lines end in CRLF, with a fresh cache: `tick doctor` reports `No issues found`, as it does for the same store with LF endings.
- An otherwise valid store whose file ends `…}\n\r`: `tick list` succeeds, and a following `tick doctor` reports `No issues found`. Both treat the final `\r` as an empty, skipped line.
- Doctor reads `tasks.jsonl` only through the shared line reader. No reader of `tasks.jsonl`, in doctor or in the CLI's doctor command, keeps its own scanner or a line-length ceiling.

STATUS: complete

SPEC CONTEXT: §1.2 names doctor's second defect: its own bufio.Scanner hit the same 64 KiB ceiling, and doctor never checked the scanner's error, so it judged the prefix before a long line as if it were the whole store. That produced false passes and false orphan errors. §2.1/§2.2 require the store and doctor to read tasks.jsonl through one shared line reader with the store's line definition: "\n" ends a line, one "\r" before it (or at EOF) is stripped, only a line that is empty after the terminator is removed is skipped, whitespace-only lines are not skipped, and numbering is 1-based with skipped lines counted. §5.4: once the whole file is read, a mid-file oversized line produces no false orphan error. §8.1/§8.4/§8.6 set the tests, including revisiting TestScanJSONLines and TestJsonlSyntaxCheck's whitespace-only cases.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/doctor/jsonl_reader.go:46-75 — `ScanJSONLines` now reads through `jsonl.Lines(f)` (:56). No bufio.Scanner is left and the TrimSpace skip is gone. `LineNum` comes from the shared reader's `l.Num` (:62). A reader error is returned wrapped instead of being dropped (:57-59).
  - internal/jsonl/lines.go:29-53 — the shared reader, which `ParseJSONL` in the store also uses (internal/storage/jsonl.go:95). Doctor and store therefore share one line definition, including the at-EOF "\r" strip (Corrigendum 2026-09-28).
  - internal/cli/doctor.go:28-29 — `RunDoctor`'s pre-scan goes through `doctor.ScanJSONLines` and passes the result to every check via `WithScan`.
  - internal/doctor/jsonl_reader.go:90-103 — the per-check fallback `getJSONLines` / `getTaskRelationships` routes to `ScanJSONLines`.
  - internal/doctor/task_relationships.go:84-91 — `ParseTaskRelationships` routes to `ScanJSONLines`.
  - Every line-consuming check (JsonlSyntax, IdFormat, DuplicateId, DuplicateSeq, OrphanedParent, OrphanedDependency, SelfReferentialDep, DependencyCycle, ChildBlockedByParent, ParentDoneWithOpenChildren) reads through `getJSONLines` / `getTaskRelationships`. The Cache check (internal/doctor/cache_staleness.go:28) reads the raw bytes with os.ReadFile only to hash them. It does not read lines, so it has no scanner and no ceiling (§5.5 leaves it unchanged).
- Notes: Every criterion was settled by reading. Doctor's Cache check hashes the raw file bytes, and the store's `readAndEnsureFresh` (internal/storage/store.go:359-375) hashes the same `os.ReadFile` bytes. So a cache that `tick list` builds is fresh for doctor, CRLF and trailing-"\r" files included. That makes the `No issues found` criteria reachable. The whitespace-only line now reaches the JSONL check and fails it with the loader's reason (internal/doctor/jsonl_syntax.go:37-45, delivered by the follow-on Task 2-2). The relationship checks keep skipping a line whose Parsed is nil or that has no string id (internal/doctor/task_relationships.go:28-39), as §5.4 requires. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - AC1: internal/cli/doctor_line_reading_test.go:116-127. A 70,000-byte line sits mid-file. tick-aaa111 has parent tick-ddd444 and tick-bbb222 is blocked by tick-eee555, both after the long line. The cache is built by `list`, and the test asserts exit 0 and `No issues found`. The test would fail if the read stopped at the long line, because orphan errors would appear.
  - AC2: doctor_line_reading_test.go:129-133 plus the `assertDoctorAgreesWithList` helper (:83-95). It asserts that list exits 1 naming line 3, that doctor exits 1, and that doctor's JSONL failures are exactly [3].
  - AC3: doctor_line_reading_test.go:135-146 runs the CLI over a mix of CRLF, empty and whitespace-only lines, expecting line 6 from both list and doctor. internal/doctor/jsonl_reader_test.go:179-204 compares ScanJSONLines line for line with the shared reader on a mixed fixture that ends without a newline.
  - AC4: doctor_line_reading_test.go:148-152 checks an LF store and its CRLF twin, each with a fresh cache.
  - AC5: doctor_line_reading_test.go:154-158. `setupListedProject` requires list to exit 0 (:36-38), then the test asserts `No issues found`.
  - AC6 is structural and was verified by reading. Behavioural guards: jsonl_reader_test.go:142-161 reads a line over 64 KiB plus the line after it, and :163-177 covers CRLF stripping and the skipped final lone "\r".
  - §8.6 revisits are done. jsonl_reader_test.go:125-140 replaces the old whitespace-skip case: whitespace-only lines are kept, numbered, and have nil Parsed. jsonl_syntax_test.go:59-77 replaces the "only whitespace-only lines passes" case: each whitespace-only line fails and is named by number.
- Notes: Tests are focused. The CLI tests observe the user-visible verdict and the unit tests pin the reader contract, without redundant restatement.

CODE QUALITY:
- Project conventions: Followed. This matches CLAUDE.md's rule to read tasks.jsonl through internal/jsonl and never bufio.Scanner. Errors are wrapped with %w. Tests use the stdlib only, with t.Run and t.Helper.
- SOLID principles: Good. Line definition lives in one place (jsonl.Lines); doctor adds only JSON-map parsing on top.
- Complexity: Low
- Modern idioms: Yes. It uses range-over-func with iter.Seq2.
- Readability: Good. Comments on ScanJSONLines and ParseTaskRelationships hold true against the code.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
