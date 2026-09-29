TASK: Store Read Errors Name The Line And The Task (oversized-text-field-bricks-the-store-1-6, tick-2d2b01)

ACCEPTANCE CRITERIA:
- A store whose line 2 holds task `tick-a1b2c3` with a wrong-typed field (`"priority":"high"`): `tick list` exits 1 with an error naming line 2, `tick-a1b2c3` and the decoder's reason (§3, §8.2)
- Same store: a write such as `tick create` and `tick rebuild` each fail with the same line, task ID and reason (§3)
- A line carrying a string `id` whose `created` timestamp does not parse: the error names the line, that ID and the timestamp reason (§3)
- A malformed-JSON line or a whitespace-only line: the error names the line by number alone, with no task ID, followed by the reason (§3, §8.2)
- A line that fails to load and carries no string `id` (missing `id`, e.g. `{"title":"T"}`; non-string `id`; not a JSON object, e.g. `null`, `[]`): the error names the line by number alone, followed by the reason (§3)
- An empty line 2 followed by a wrong-typed task on line 3: the error names line 3, counting the skipped line (§2.2, §3)

STATUS: complete

SPEC CONTEXT: §3 requires every error from reading tasks.jsonl into the store to name the failing line (1-based, numbered per §2.2 with skipped empty lines counted) and, when the line is a JSON object whose `id` is a string, that task ID; otherwise the line alone. The loader's own reason follows, e.g. `failed to parse tasks.jsonl: line 2 (tick-a1b2c3): json: cannot unmarshal string into Go struct field taskJSON.priority of type int`. Loading includes timestamp parsing in `Task.UnmarshalJSON`, so `null` and `{}` fail on the empty `created`. The purpose is hand-edit recovery. §8.2 and §8.6 require the error tests and a revisit of `TestParseJSONL`.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/storage/jsonl.go:89-110 (`ParseJSONL` routes each load failure through `lineError`), internal/storage/jsonl.go:120-130 (`lineError`: probes the line into `struct{ ID *string }` and adds the ID only when the probe decodes and the id is a non-null string), internal/storage/jsonl.go:112-118 (`DecodeTaskLine`, the loader, whose error passes through unchanged via `%w`). The store's `failed to parse tasks.jsonl: %w` wraps are at internal/storage/store.go:166 (`ReadTasks`), :238 (`Rebuild`) and :367 (`readAndEnsureFresh`), so every command sees the same message.
- Notes: The probe handles every case in the spec correctly. Malformed JSON, trailing data and whitespace-only lines fail the probe's syntax check, so they get the line alone. `[]` and a bare JSON string fail with a type error, so they get the line alone. `null`, a missing `id` and `"id":null` leave the pointer nil, so they get the line alone. A numeric `id` fails with a type error, so it gets the line alone. A string `id` gets the line and the ID, whatever other field fails, because the probe ignores every other field. The shared reader's read-failure path (internal/storage/jsonl.go:97 wrapping `read line N: ...` from internal/jsonl/lines.go:35) also names the line, so §3's "every error" holds even on the path a `bytes.Reader` can't reach. No stale references to the old `failed to parse line` wording remain in internal/, cmd/ or README.md.

TESTS:
- Status: Adequate
- Coverage: Storage level (internal/storage/jsonl_test.go:631-694): the whitespace-only line after a skipped empty line (line 3), malformed JSON by line alone, a wrong-typed field with line and ID (exact message), an unparseable `created` with line and ID, and seven no-string-id shapes by line alone (`{"title":"T"}`, `"id":42`, `"id":null`, `null`, `[]`, a bare string, a truncated object). A skipped empty line is counted. `assertParseError` takes the expected reason from the real loader rather than hard-coding it, and it fails fast if a fixture unexpectedly loads. CLI level (internal/cli/store_read_errors_test.go:20-72): runs through `App.Run` and asserts the full stderr and exit 1 for `list`. It asserts the same message for `create` then `rebuild` on the same store, plus the timestamp, malformed-JSON, whitespace-only, no-string-id and empty-line-2 cases. The old `it returns error for invalid JSON in bytes` test was revisited and now asserts the exact line-alone message (§8.6). internal/cli/store_line_reading_test.go:100 was updated to the new wording.
- Notes: Each assertion compares the exact error string, so dropping the ID, dropping the line number, adding an ID where none belongs, or miscounting lines would each fail a test. The storage-level and CLI-level cases overlap on purpose: the first pins `ParseJSONL`, the second pins what every command surfaces through the store wrap. That overlap is not bloat.

CODE QUALITY:
- Project conventions: Followed (stdlib testing, `t.Run` subtests named "it ...", `t.Helper` on helpers, `%w` wrapping)
- SOLID principles: Good (`lineError` has one job; the loader's reason stays separate from the line naming)
- Complexity: Low
- Modern idioms: Yes (range-over-func iterator from the shared reader)
- Readability: Good; the comments on `lineError`, `DecodeTaskLine` and `assertParseError` match the code
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
