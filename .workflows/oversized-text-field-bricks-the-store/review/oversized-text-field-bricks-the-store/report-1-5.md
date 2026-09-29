TASK: Beads Importer Reads Issue Lines Of Any Length (oversized-text-field-bricks-the-store-1-5, tick-4b2101)

ACCEPTANCE CRITERIA:
- A `.beads/issues.jsonl` holding an issue line longer than 64 KiB among ordinary issues: `tick migrate --from beads` reads that line and completes the import, and every issue is imported, the long one included. Today the import aborts before anything is written (§2.3, §8.1)
- A `.beads/issues.jsonl` with whitespace-only lines among its issues: those lines are skipped as they are today, not reported as failed entries, because the importer trims each line (§2.3)
- The importer imposes no per-line size limit, neither Go's default scanner limit nor any larger one set in its place (§2.1, §2.3)

STATUS: complete

SPEC CONTEXT: §2.1 removes every read-side per-line size limit, and names `internal/migrate/beads/beads.go`'s `bufio.NewScanner` as one of the three production scanners in scope. §2.3 says the beads importer reads an external file, so it does not use the shared `internal/jsonl` reader and keeps its own line rules (each line trimmed). It drops the ceiling all the same. §8.1 requires a test that the importer reads an `issues.jsonl` line past 64 KiB. Per the task context, the imported long issue must also stay inside Phase 3's migrate caps (description <= 50,000 characters, title <= 500 characters on one line). It also depends on Task 1-1's reader to reopen a store whose resulting record passes 65,536 bytes.

IMPLEMENTATION:
- Status: Implemented
- Location: internal/migrate/beads/beads.go:85-103 (`BeadsProvider.Tasks` read loop), internal/migrate/beads/beads.go:108-117 (`parseIssueLine`)
- Notes:
  - The `bufio.Scanner` is replaced by `bufio.NewReader(file)` plus `ReadString('\n')` (beads.go:86, :89). `ReadString` grows its result to fit the whole line, and no buffer size or maximum is set anywhere. That meets AC3: neither the default token limit nor a larger chosen one applies.
  - The line rules are kept. Each raw line is passed through `strings.TrimSpace`, and a line that is empty after trimming is skipped (beads.go:94). The trailing `\n` and any `\r` that `ReadString` keeps are removed by the same trim. The result is identical to the old `scanner.Text()` then `TrimSpace` path, including CRLF lines and a final line with no newline: a final chunk returned with `io.EOF` is processed before the loop breaks (beads.go:94-100).
  - The file does not import the shared `internal/jsonl` reader, as §2.3 requires.
  - A non-EOF read error still returns an error naming the file (beads.go:90-92). `Engine.Run` discards the task slice whenever `Tasks()` returns an error (internal/migrate/engine.go:61-64), so the partial slice returned with it has no effect.
  - The comments on `Tasks` (beads.go:63-67) and `parseIssueLine` (beads.go:106-107) match the code.

TESTS:
- Status: Adequate
- Coverage:
  - AC1, provider level: `TestBeadsProviderLineLength` (internal/migrate/beads/beads_test.go:582) places an issue line of exactly 65,535 bytes, 65,536 bytes, 1 MiB and 16 MiB between two ordinary issues. It checks all three titles in order, and checks the long issue's description byte for byte. `paddedIssueLine` (beads_test.go:572) builds lines of an exact byte count.
  - AC1, end to end: `TestMigrateBeadsLongIssueLine` (internal/cli/migrate_test.go:778) runs `tick migrate --from beads` on a source whose middle line is built with `json.Marshal` from 12,000 `<` characters. That is 12,000 characters, well inside the Phase 3 cap, and at least 65,536 bytes once escaped; the test guards the length at migrate_test.go:790. The test asserts exit 0, `Done: 3 imported, 0 failed`, all three issues persisted, and the long description stored in full. `assertListedIDs` then confirms the store reopens with every task.
  - AC2: `TestBeadsProviderWhitespaceLines` (beads_test.go:618) mixes space-only, tab-only, `"   \r\n"`, a CRLF-terminated issue, an empty line and a final whitespace-only line with no newline. It asserts that exactly the two real issues come back. If a whitespace-only line were emitted as a malformed sentinel entry, the count assertion would fail. The existing blank-lines test (beads_test.go:269) still holds.
  - AC3: the 16 MiB case fails against any plausible chosen larger limit, as well as against the default one.
- Notes: The provider-level and CLI-level tests do not duplicate each other. The first pins the reader boundary cases. The second pins that the import completes, what it persists, and that the store reopens afterwards. No existing beads test was weakened.

CODE QUALITY:
- Project conventions: Followed (stdlib `testing`, `t.Run` subtests, `t.Helper()` on helpers, `t.TempDir()`, `fmt.Errorf` with `%w`; tasks.jsonl-only reader rule in CLAUDE.md correctly not applied to the external issues.jsonl)
- SOLID principles: Good — line reading and per-line parsing (`parseIssueLine`) are separated
- Complexity: Low
- Modern idioms: Yes — `errors.Is(readErr, io.EOF)`, `new(expr)` already in use
- Readability: Good
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
