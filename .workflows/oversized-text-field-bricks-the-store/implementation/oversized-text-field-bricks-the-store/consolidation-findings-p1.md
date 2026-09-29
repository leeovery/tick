# Consolidation Findings: oversized-text-field-bricks-the-store (Phase 1)

## Findings

### F1: CLAUDE.md's package map omits the shared tasks.jsonl line reader
- **Class**: drift
- **Failure**: CLAUDE.md's Architecture block is the orientation map every agent session loads, and it lists each `internal/` package down to `internal/testutil/`. It does not list `internal/jsonl/`, which Phase 1 added. An agent writing a new reader of `tasks.jsonl` (an export, a new diagnostic) orients from that map, never learns the ceiling-free reader exists, and uses `bufio.NewScanner`. That brings back the 64 KiB ceiling for that reader. Nothing flags it: the phase's regression guards cover only the store's path through `ParseJSONL`. The failure appears only when a store holds a line of 65,536 bytes or more and the new reader rejects it or stops short, the same defect this work removes.
- **Evidence**: `CLAUDE.md:31-36` (package list; `internal/storage/` at `:33` is the natural neighbour); `internal/jsonl/lines.go:1-53` (package introduced by 4e98f8fc, task 1-1); `internal/storage/jsonl.go:95` (the store's use of it)
- **Proposed shape**: Add one line to the Architecture block after `internal/storage/`, e.g. `internal/jsonl/            → tasks.jsonl line reader with no line-length ceiling; read tasks.jsonl through it, never bufio.Scanner`. Leave out any claim about which packages read through it ("store and doctor"). That keeps the line true now, while doctor keeps its own scanner until Phase 2 task 2-1, and afterwards with no second edit. This is documentation only, with no code or test change.
- **Bank**: reviewer (task 1-1): "CLAUDE.md Architecture block does not list the new internal/jsonl shared line reader package". Confirmed against the final state: `CLAUDE.md` still has no `internal/jsonl` entry. The entry suggested waiting for the Phase 2 boundary; the wording above removes the need to wait.
