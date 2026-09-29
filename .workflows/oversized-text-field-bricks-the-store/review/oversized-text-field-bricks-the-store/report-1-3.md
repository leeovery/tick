TASK: Writes That Take Their Own Task Past The Old Ceiling Report Success (oversized-text-field-bricks-the-store-1-3, tick-dba7a7)

ACCEPTANCE CRITERIA:
- A task whose line is under 65,536 bytes: `tick update <id> --description` with text that takes the line to 65,536 bytes or more exits 0 with its normal task detail output, carrying the new description (§1.1, §7)
- `tick create` with a description that makes the new task's line 65,536 bytes or more exits 0 with its normal task detail output (§1.1, §7)
- A task gains notes one `tick note add` at a time, each note within the 2,000-character cap. The `note add` that takes its line to 65,536 bytes or more exits 0 with its normal task detail output, and the new note is included (§1.2, §2.4, §7)
- After each of these writes, every following command opens the store: a read (`tick list`, `tick show <id>`), a further write to the grown task and `tick rebuild` all succeed (§2.1)

STATUS: complete

SPEC CONTEXT: §1.1 records that create/update/note add commit an over-ceiling write and then exit 1 outside --quiet because their read-back (outputMutationResult -> queryShowData) goes through a reader capped at 64 KiB. §7 says the post-write signal needs no change of its own: once every reader accepts a line of any length (§2.1, Task 1-1), the read-back succeeds. §1.2/§2.4 note that note counts are unbounded, so accumulated in-cap notes can cross the line. §6.1's 50,000-character description cap (Phase 3) means description scenarios must cross 65,536 bytes via escape inflation while staying under 50,000 characters.

IMPLEMENTATION:
- Status: Implemented
- Location: No production change belongs to this task (§7); the task commit (492ced49) adds only internal/cli/over_ceiling_write_test.go. The behaviour rests on Task 1-1's reader: internal/jsonl/lines.go:29 (bufio.Reader.ReadBytes, no length limit), consumed by internal/storage/jsonl.go:95 (ParseJSONL), which every store read uses (internal/storage/store.go:164, :236, :365). The write path serialises through internal/storage/jsonl.go:23 -> internal/task/task.go:106 (json.Marshal, HTML escaping on), and the read-back is internal/cli/helpers.go:23.
- Notes: The whole path from committed write to read-back is ceiling-free as read. The planned route (verify via tests, no code change) matches the spec's §7 intent.

TESTS:
- Status: Adequate
- Coverage: internal/cli/over_ceiling_write_test.go:60 TestWritesCrossingOldCeiling.
  - update (:67): asserts the seed line is under 65,536 bytes (:69), runs `update --description` with 11,000 '<' characters (66,000 escaped bytes, well under the 50,000-character cap), decodes the detail document and checks id and the full description (:76). It then asserts that the stored line is at least 65,536 bytes (:77).
  - create (:81): same inflation. Asserts title and description in the detail output (:88) and a stored line of at least 65,536 bytes (:89).
  - note add (:93): adds notes of 1,999 '<' plus one letter each (exactly 2,000 characters, which ValidateNoteText in internal/task/notes.go:57 accepts). Each add runs through runToonCommand, which fails on any non-zero exit (internal/cli/toon_decode_test.go:123). On the add that crosses, it checks the id and that every note row, the new one included, matches in order (:110-117). A 20-add bound fails the test if the line never crosses (:121).
  - Criterion 4 is covered for all three writes by assertStoreOperable (:44): list, show, `update --title` of the grown task, rebuild, then list again.
  - "Exits 0" is enforced by runToonCommand. None of the runs use --quiet, so the read-back path from §1.1 is exercised. If a read ceiling came back, every scenario would fail at the read-back.
- Notes: Focused and not redundant. Each assertion maps to a criterion. The helpers storedLineLen and assertLineAtLeastOldCeiling are shared with Task 1-4's growth tests (internal/cli/over_ceiling_growth_test.go:33, :79) rather than duplicated.

CODE QUALITY:
- Project conventions: Followed. Uses stdlib testing, "it ..." subtest names, t.Helper on helpers, and decoded-TOON value assertions (decodeToonDoc, toonRows, assertToonFields).
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes (strings.SplitSeq, range over int)
- Readability: Good. The escapeInflated comment (:15-16) is accurate: json.Marshal HTML-escapes '<' to the 6-byte <.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
