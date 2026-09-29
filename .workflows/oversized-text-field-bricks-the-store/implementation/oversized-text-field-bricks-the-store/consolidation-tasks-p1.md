# Consolidation Tasks: Oversized Text Field Bricks The Store (Phase 1)

## Task 1: Corrections
placement: phase 1
severity: corrections

**Problem**: CLAUDE.md's Architecture block (`CLAUDE.md:31-36`) lists every `internal/` package except `internal/jsonl/`, the ceiling-free `tasks.jsonl` line reader Phase 1 added. An agent adding a new reader of `tasks.jsonl` orients from that map, never finds the shared reader, and reaches for `bufio.NewScanner` — reintroducing the 64 KiB ceiling for that reader. Nothing catches it until a store holds a line of 65,536 bytes or more and the new reader rejects it. As the tree stands, the block runs from `cmd/tick/main.go` (`CLAUDE.md:30`) through `internal/testutil/` (`:36`) with no `internal/jsonl` entry. The package is `internal/jsonl/lines.go:1-53` (`jsonl.Lines`, added by 4e98f8fc in task 1-1). The store reads through it at `internal/storage/jsonl.go:95`. Doctor still has its own `bufio.NewScanner` at `internal/doctor/jsonl_reader.go:37`. The phase's regression guards cover only the store's path through `ParseJSONL`, so no test catches a new reader that bypasses the package.
**Solution**: One edit — `CLAUDE.md:33`: add a line after `internal/storage/` in the Architecture block, e.g. `internal/jsonl/            → tasks.jsonl line reader with no line-length ceiling; read tasks.jsonl through it, never bufio.Scanner`. The line names no consumers (derived from the finding: doctor keeps its own scanner until Task 2-1, so a consumer list would be false now or need a second edit after Phase 2). Documentation only; no code or test change.

**Outcome**: CLAUDE.md's Architecture block lists `internal/jsonl/` directly after `internal/storage/`. The entry calls it the ceiling-free `tasks.jsonl` line reader and tells any new reader of `tasks.jsonl` to go through it rather than `bufio.Scanner`. No Go source or test changes.

**Acceptance Criteria**:
- [ ] Reading CLAUDE.md's Architecture block, an agent finds an `internal/jsonl/` entry on the line immediately after `internal/storage/`, with its `→` in the same column as the neighbouring entries
- [ ] That entry calls the package the `tasks.jsonl` line reader with no line-length ceiling and directs new readers of `tasks.jsonl` through it, never `bufio.Scanner`
- [ ] The entry names no consuming packages, so it stays true while doctor keeps its own scanner and after Phase 2 task 2-1, with no second edit
- [ ] No other line of CLAUDE.md changes, no file other than CLAUDE.md changes, and `go test ./...` passes unchanged

**Do**:
- `CLAUDE.md`: insert one line into the Architecture code block between `internal/storage/` (`:33`) and `internal/doctor/` (`:34`), worded as the Solution's example or close to it
- Leave out any list of which packages read through `internal/jsonl` (such as "store and doctor")
- Documentation only: edit no Go source, test, or other file
