TASK: Corrections (oversized-text-field-bricks-the-store-1-8, tick-ce257a) — add the shared `internal/jsonl/` line reader to CLAUDE.md's Architecture block

ACCEPTANCE CRITERIA:
- Reading CLAUDE.md's Architecture block, an agent finds an `internal/jsonl/` entry on the line immediately after `internal/storage/`, with its `→` in the same column as the neighbouring entries
- That entry calls the package the `tasks.jsonl` line reader with no line-length ceiling and directs new readers of `tasks.jsonl` through it, never `bufio.Scanner`
- The entry names no consuming packages, so it stays true while doctor keeps its own scanner and after Phase 2 task 2-1, with no second edit
- No other line of CLAUDE.md changes, no file other than CLAUDE.md changes, and `go test ./...` passes unchanged

STATUS: complete

SPEC CONTEXT: §2.2 requires the store and doctor to read `tasks.jsonl` through one shared line reader so "no future reader of the store can reintroduce a ceiling". This task came from Phase 1 consolidation finding F1 (implementation/.../consolidation-findings-p1.md): CLAUDE.md's package map did not list `internal/jsonl/`, so an agent orienting from it would write a new `tasks.jsonl` reader with `bufio.NewScanner` and bring the 64 KiB ceiling back. The fix is documentation only.

IMPLEMENTATION:
- Status: Implemented
- Location: CLAUDE.md:34 (`internal/jsonl/           → tasks.jsonl line reader with no line-length ceiling; read tasks.jsonl through it, never bufio.Scanner`), directly after `internal/storage/` at CLAUDE.md:33 and before `internal/doctor/` at CLAUDE.md:35
- Notes:
  - Position: the entry sits on the line immediately after `internal/storage/`, as required.
  - Alignment: `internal/jsonl/` (15 chars) plus 11 spaces puts `→` at column 27. That matches `internal/storage/` (17 + 9), `internal/doctor/` (16 + 10) and `internal/testutil/` (18 + 8) at CLAUDE.md:33, :35 and :37.
  - Wording: this is the Solution's example word for word. It calls the package the ceiling-free `tasks.jsonl` line reader and tells new readers to go through it rather than `bufio.Scanner`.
  - No consumers named: the entry lists no packages that read through it. That keeps it true in the final tree, where both the store (internal/storage/jsonl.go:95) and doctor (internal/doctor/jsonl_reader.go:56) read through `jsonl.Lines`, with no second edit needed after task 2-1.
  - The claim holds against the code: `jsonl.Lines` (internal/jsonl/lines.go:29-48) reads with `bufio.NewReader`/`ReadBytes`, not `bufio.Scanner`, and imposes no line-length limit. The package doc comment at internal/jsonl/lines.go:1-2 says the same thing the entry does.
  - I found no other line in CLAUDE.md that carries content from this feature. The Architecture neighbours are unchanged in substance.

TESTS:
- Status: Adequate (N/A — documentation-only task)
- Coverage: The plan forbids code or test changes for this task ("Documentation only: edit no Go source, test, or other file"), so no new test is expected. Nothing under test depends on the Architecture block's content.
- Notes: None

CODE QUALITY:
- Project conventions: Followed. The entry uses the block's `path → description` shape and the shared arrow column.
- SOLID principles: N/A (no code)
- Complexity: Low
- Modern idioms: N/A
- Readability: Good. The entry states both what the package is and the rule a new reader must follow.
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "No other line of CLAUDE.md changes, no file other than CLAUDE.md changes, and `go test ./...` passes unchanged" — reading the current tree cannot show the task's own diff or a suite result. To settle it, inspect the task 1-8 commit's diff (for example `git show --stat` and `git show -- CLAUDE.md` for that commit) to confirm it adds only CLAUDE.md:34 and touches no other file, and run `go test ./...`.
