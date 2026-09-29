TASK: JSONL Syntax Check Judges Each Line By Loading It As A Task (oversized-text-field-bricks-the-store-2-2, tick-9ab6e2)

ACCEPTANCE CRITERIA:
- A two-task store with one added line, where that line is whitespace-only, `null`, `[]`, `{}`, malformed JSON, a wrong-typed field such as `"priority":"high"`, or a `created` timestamp that does not parse: `tick doctor` exits 1 with a `JSONL syntax` failure that names that line and gives the loader's reason in place of today's `invalid JSON`. `tick list` on the same store fails, naming the same line (§5.1, §5.2, §8.4)
- A store with several lines that fail to load: the JSONL check reports one failure per line, each naming its own line (§5.1)
- A line that loads but carries a status or type outside the allowed values, or a priority outside 0–4: the JSONL check passes it, and doctor adds no enum or range check (§5.1)
- A line that loads but repeats a value within its own `tags`, `refs` or `blocked_by` list: the JSONL check passes it, and doctor adds no check for the repeat (§5.1, Corrigendum 2026-09-28 on §5.1)
- A valid store with a fresh cache: `tick doctor` reports `No issues found` (§5.1, §8.4)
- The check keeps its name and its suggestion. It is titled `JSONL syntax` in doctor's output whether it passes or fails, and each failure's suggestion stays a hand fix of the named line. The README's doctor section (its "Checks for:" list at `README.md:396` and the sample tested against it) and `tick help doctor` are unchanged (§5.2)
- A line that is not a JSON object (such as `[]` or `null`) or whose `id` is not a string: the relationship and hierarchy checks skip it as they do today, and the JSONL check reports it (§5.4)

STATUS: issues_found

SPEC CONTEXT: §1.2/§1.4 — doctor's readability test was looser than the store's (json.Valid vs. loading as a task), so `null`, `[]`, `{}`, wrong-typed fields and bad timestamps passed doctor while every command failed; agents treat a passing doctor as "every command can open the store". §5.1 requires every line the shared reader returns to load exactly as commands load it (same decoding, timestamp parsing included), one failure per line naming the line and giving the loader's reason; enum/range validation and in-list repeats (Corrigendum 2026-09-28) stay outside doctor. §5.2 keeps the `JSONL syntax` name, README list and help text, and the hand-fix suggestion; only the failure detail changes. §5.4 keeps relationship/hierarchy checks skipping non-object / non-string-id lines. §8.6 names `TestJsonlSyntaxCheck` for revisiting.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/storage/jsonl.go:112-118 — `DecodeTaskLine` extracted from `ParseJSONL`; `ParseJSONL` now calls it at internal/storage/jsonl.go:100, so the store and doctor share one decode (json.Unmarshal into `task.Task`, whose `UnmarshalJSON` at internal/task/task.go:109 parses created/updated/closed).
  - internal/doctor/jsonl_syntax.go:19-33 — `Run` returns `loadFailures(lines)` when non-empty, else a single passing `JSONL syntax` result.
  - internal/doctor/jsonl_syntax.go:37-45 — `loadFailures` decodes each `JSONLine.Raw` through `storage.DecodeTaskLine`, one result per failing line.
  - internal/doctor/jsonl_syntax.go:47-59 — `loadFailure`: Name `JSONL syntax`, SeverityError, Details `Line N: <loader reason> — <80-byte preview>`, Suggestion `Manual fix required` (unchanged).
- Notes: The lines doctor judges come from `ScanJSONLines` over the shared `jsonl.Lines` reader (internal/doctor/jsonl_reader.go:56), the same line definition `ParseJSONL` iterates (internal/storage/jsonl.go:95), and `Raw` is the byte-identical `l.Text`, so doctor fails exactly the lines the store's decode fails, numbered the same way. Doctor reports every failing line where the store stops at the first — which is what §5.1 asks. README.md and internal/cli/help.go have no diff across the whole feature (README.md:396 "Checks for: JSONL syntax errors, …" and help.go:217 intact). `taskRelationshipsFromLines` (internal/doctor/task_relationships.go:28-39) is untouched and still skips nil-Parsed and non-string-id lines. The old `json.Valid` path and its `invalid JSON` detail are gone; no production text still says `invalid JSON`. No drift from the plan.

TESTS:
- Status: Adequate
- Coverage:
  - internal/doctor/jsonl_syntax_test.go:335-380 — all seven AC1 line kinds (whitespace-only, null, [], {}, malformed, wrong-typed priority, unparseable created) at line 3 of a two-task store, asserting the full CheckResult: name, severity, `Line 3: <exact loader reason> — <line>`, and `Manual fix required`.
  - internal/doctor/jsonl_syntax_test.go:382-411 — status/type outside the allowed values, priority 9 and -1, repeated tag/ref/blocker, and an unknown field each yield exactly `[{JSONL syntax, Passed}]`.
  - internal/cli/doctor_line_reading_test.go:160-188 — the same seven line kinds through `tick doctor` and `tick list`: list exits 1 naming line 3, doctor exits 1 with exactly one JSONL failure at line 3, and doctor's detail carries list's own reason text verbatim (`listFailureReason`), so the agreement is observed rather than restated.
  - internal/cli/doctor_line_reading_test.go:190-208 — several failing lines give JSONL failures at exactly [2 4 5].
  - internal/cli/doctor_line_reading_test.go:210-241 — enum/range/repeat lines: `✓ JSONL syntax: OK` and no failure but the cache's, i.e. doctor adds no enum, range or repeat check.
  - internal/cli/doctor_line_reading_test.go:243-245 — valid store with a fresh cache ends in `No issues found.`
  - internal/cli/doctor_line_reading_test.go:247-274 — `[]`, `null` and `"id":7` (with a dangling parent and blocker) are JSONL failures at [2 3 4] while all six relationship/hierarchy checks pass, which they would not if those lines were not skipped.
  - Name retention on pass and fail: internal/doctor/jsonl_syntax_test.go:215-257; hand-fix suggestion: :185-198 and the full-struct assertions above.
  - internal/cli/doctor_test.go fixtures were given `created`/`updated` so tests that expect a passing JSONL check or `No issues found` use lines that load (no remaining `{"id"…}` fixture there lacks `created`).
  - The spec-named `it does not validate JSON field names or values — only syntax` was replaced by the two table tests above.
- Notes: Unit and CLI tables cover the same line kinds but assert different properties (exact loader text vs. agreement with `tick list` and absence of extra doctor checks); not redundant. Each assertion would fail if the check reverted to `json.Valid` or diverged from the store's decode.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, t.Run "it …" subtests, t.Helper on helpers, error text via %v in details, no testify)
- SOLID principles: Good — one decode function (`storage.DecodeTaskLine`) is the single definition of "loads as a task" for both store and doctor; the check stays a thin consumer.
- Complexity: Low
- Modern idioms: Yes
- Readability: Good
- Issues: One stale subtest name (see FINDINGS).

BLOCKING ISSUES:
- None

FINDINGS:
- [in-scope] [contained] internal/doctor/jsonl_syntax_test.go:14 — the subtest is still named "it returns passing result when all lines are valid JSON", the old `json.Valid` contract. This task replaced that contract, and the same file now fails valid-JSON lines (`null`, `[]`, `{}`, wrong-typed fields at :342-349). Rename it to state the rule the fixture actually exercises, e.g. "it returns passing result when every line loads as a task". The remedy is test-name text only, so it is non-blocking. — FAILS: the suite states a pass condition the check falsifies. A reader taking the check's contract from `TestJsonlSyntaxCheck` would conclude any syntactically valid JSON line passes, which is exactly the false "healthy" this task removed.

UNSETTLED:
- None
