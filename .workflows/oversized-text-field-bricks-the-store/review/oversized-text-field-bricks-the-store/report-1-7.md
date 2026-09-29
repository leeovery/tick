TASK: Tick Rebuild Keeps The Cache Until It Can Build A New One (oversized-text-field-bricks-the-store-1-7, tick-f986bf)

ACCEPTANCE CRITERIA:
- A store that fails to parse (for example, a wrong-typed field on one line) with an existing `cache.db`: `tick rebuild` exits 1 with the parse error naming the line, and `cache.db` is byte-for-byte unchanged (§4, §8.3)
- `tick rebuild --verbose` on a valid store logs `reading JSONL` before `deleting cache.db` (§4, §8.3)
- A valid store whose `cache.db` is corrupt: `tick rebuild` succeeds, and the rebuilt cache answers queries (§4, §8.3)

STATUS: complete

SPEC CONTEXT: §4 requires `tick rebuild` to read and parse `tasks.jsonl` before it touches `cache.db`. A failed parse exits 1 with the §3 parse error (line number, plus task ID when the line carries a string `id`) and leaves `cache.db` exactly as it was. Only a successful parse removes the old cache and builds a fresh one. Rebuild still recovers from a corrupt `cache.db`, and the verbose log order becomes `reading JSONL` then `deleting cache.db`. §8.3 asks for a byte-for-byte cache check on parse failure, requires the corrupt-cache store test (`store_test.go:571`) to keep passing, and requires the verbose-order store test to be updated. §8.6 lists the rebuild tests for revisiting.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/storage/store.go:222-268 (`Store.Rebuild`). The read (`:230-234`) and parse (`:236-239`) now run before the close-existing-cache step (`:241-245`), the delete (`:248-251`), `OpenCache` (`:254-258`) and `cache.Rebuild` (`:261-264`). The doc comment at `:219-221` has been updated to match.
- Notes: `RunRebuild` (internal/cli/rebuild.go:11-30) opens the store with `openStore` → `storage.NewStore` (internal/cli/helpers.go:46-52). `NewStore` does not open the cache (store.go:55-74), so nothing touches `cache.db` before `Rebuild` returns from a read or parse error. The failure returns `failed to parse tasks.jsonl: %w` around `ParseJSONL`'s `lineError` (internal/storage/jsonl.go:122-129), which names the line and the task ID. `App.Run` prints `Error: ...` and returns 1. Corrupt-cache recovery is unchanged: `s.cache` is nil, `removeCache` deletes the garbage file and `OpenCache` creates a fresh schema. A parse that succeeds but whose `cache.Rebuild` fails (for example, a duplicate value within a task's tags, per the 2026-09-28 corrigendum) still ends with the old cache deleted. §4 gates only on the parse, and that old cache has no consumer: every query path would re-run `cache.Rebuild` through `ensureFresh` and fail the same way. So this is not a defect. Help text (internal/cli/help.go:208-211) and README.md:398-403 make no claim about ordering.

TESTS:
- Status: Adequate
- Coverage:
  - Criterion 1: internal/cli/rebuild_test.go:300-341 builds a real cache, hand-edits a wrong-typed `priority` onto line 2, then asserts exit 1, stderr containing `failed to parse tasks.jsonl: line 2 (tick-bbb222)`, empty stdout, and `bytes.Equal` on `cache.db` before and after. internal/storage/store_test.go:615-667 checks the same contract at the store layer. With the old order, the post-failure `os.ReadFile(cachePath)` would hit a missing file and `t.Fatalf`, so both tests would catch a regression.
  - Criterion 2: store_test.go:734-776 asserts the exact log sequence, now `reading JSONL` then `deleting cache.db`. rebuild_test.go:280-285 adds an index-order check at the CLI layer.
  - Criterion 3: rebuild_test.go:343-371 writes a garbage `cache.db`, runs `tick rebuild` and queries the rebuilt cache for the task ID. store_test.go:572-613 is unchanged and still covers the store layer.
- Notes: The parse-failure and corrupt-cache cases are each tested at both the store and the CLI layer. This is not redundant, because each layer asserts something different: the store test checks the `Rebuild` error contract, and the CLI test checks exit code, stderr and stdout.

CODE QUALITY:
- Project conventions: Followed (stdlib `testing`, `t.Run` "it ..." subtests, `t.TempDir`-based helpers, `fmt.Errorf("...: %w")` wrapping)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes
- Readability: Good
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
