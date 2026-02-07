# Tick Implementation Comparison: V1 vs V2 vs V3

## Context

Three implementations of the same `tick-core` plan were produced using different Claude workflow approaches:

- **V1** (`implementation-v1` branch): Sequential approach — main Claude session handled all tasks directly. Fastest execution, but hit compaction limits 3-4 times.
- **V2** (`implementation-v2` branch): Agent-based approach — main session orchestrates sub-agents for coding + review agents per task. Slower but reduces context pollution.
- **V3** (`implementation-v3` branch): Agent-based approach (third attempt) — same agent-based workflow as V2.

All three implement the same plan: 5 phases, 23 tasks, covering a minimal task tracker with JSONL source of truth, SQLite cache, file locking, 3 output formats, and full CLI.

---

## Overview Numbers

| Metric | V1 (Sequential) | V2 (Agents, take-two) | V3 (Agents, v3) |
|--------|-----------------|----------------------|-----------------|
| Impl LOC | 2,892 | 3,425 | 3,662 |
| Test LOC | 5,451 | 11,610 | 12,727 |
| Total | 8,343 | 15,035 | 16,389 |
| Test:Impl | 1.88:1 | 3.4:1 | 2.97:1 |
| Plan compliance | 5/5 phases | 5/5 phases | 5/5 phases |
| Go files | 45 | 45 | 46 |

All three fully implement the plan. The difference is in **how well**.

---

## Architecture (Winner: V2, narrowly)

All three use the same 3-layer architecture (task → storage → cli, unidirectional). The differences:

| Aspect | V1 | V2 | V3 |
|--------|----|----|-----|
| Storage layout | Flat package | **Sub-packages** (jsonl/, sqlite/) | Flat package |
| Domain files | 3 files (task, transition, dependency) | 1 file (all in task.go, 357 lines) | 3 files like V1 |
| CLI entry | cli.go | app.go + separate init.go, show.go, discover.go | cli.go + separate show.go |
| Tick discovery | Built into cli.go | **Separate discover.go** with walk-up | Built into cli.go |
| Binary tests | No | **Yes** (cmd/tick/main_test.go, real binary) | No |

V2's storage sub-packages and separate `discover.go` show better separation of concerns. V1 and V3 are essentially identical architecturally.

---

## Code Quality (Winner: V2)

### SQL Safety — Critical Difference

**V1** uses string interpolation for SQL:
```go
conditions = append(conditions, fmt.Sprintf("t.status = '%s'", status))
```
Input is validated upstream, but the **pattern** is dangerous and fragile.

**V2** uses parameterized queries:
```go
// Uses ? placeholders with args slice
```
This is the correct approach.

**V3** uses exported SQL constants (`ReadyCondition`/`BlockedCondition`) for the ready/blocked queries (excellent DRY pattern).

### DRY Patterns — Ready/Blocked Queries

| V1 | V2 | V3 |
|----|----|----|
| Duplicated between ready.go and stats.go | `readyWhere`/`blockedWhere` fragments reused across list/stats | **`ReadyCondition`/`BlockedCondition` exported constants** reused by 5 code paths |

V3 has the best pattern here. V1 has the worst (fragile duplication).

### Unique Weaknesses Per Version

**V1**: SQL string interpolation, duplicated ready query, double freshness check (IsFresh then EnsureFresh both hash), case-sensitivity inconsistency in domain validation.

**V2**: `FormatStats` uses `interface{}` instead of concrete type (runtime type assertion needed), `truncateTitle` uses byte count not rune count (Unicode truncation bug), incomplete usage text, `StubFormatter` in production code, `log.Printf` bypasses Logger interface.

**V3**: Double file read per Store operation (reads JSONL twice), `Store.Mutate` writes directly to `os.Stderr` (bypasses App, untestable), create/update output shows empty blocked_by/children even when just set, no --help/-h flags.

---

## Test Quality (Winner: V2, clearly)

This is where the agent-based approaches show their biggest advantage.

| Aspect | V1 | V2 | V3 |
|--------|----|----|-----|
| Test:Impl ratio | 1.88:1 | **3.4:1** | 2.97:1 |
| Format integration | 6.6k file | **832-line matrix covering ALL commands x ALL formats** | 638-line integration |
| Binary-level tests | No | **Yes** (exit codes, stderr separation) | No |
| Unicode boundary tests | Basic | **500 runes of multi-byte chars** | Present |
| Locking tests | Concurrent reads only | **All combinations** (shared/exclusive, blocked, concurrent, timeout, leak) | Lock timeout + concurrent |
| Corruption tests | Basic | **Multi-level** (garbage file, wrong schema, missing tables) | Present with self-healing |
| Spec compliance tests | No | No | **Yes** (TestToonFormatterSpecExamples, TestPrettyFormatterSpecExamples) |
| Consolidated test helpers | No (distributed) | No (distributed) | **Yes** (test_helpers_test.go) |
| Cancel-unblocks integration | Basic | **Multi-step integration** (create -> cancel -> verify ready/blocked changes) | Present |

V2's test suite is the most thorough. The 832-line format integration matrix alone provides exceptional confidence. The binary-level tests in `main_test.go` are unique to V2 and verify the actual compiled artifact. V2 also has the best locking and corruption tests.

V3 has two things V2 lacks: **spec compliance tests** (verifying exact output format against specification) and **consolidated test helpers**.

V1's tests are adequate but significantly thinner across every dimension.

---

## Unique Strengths Summary

| V1 | V2 | V3 |
|----|----|----|
| Most compact (2,892 impl lines) | Parameterized SQL queries | ReadyCondition/BlockedCondition exported constants |
| Fastest to produce | Storage sub-packages | Spec compliance formatter tests |
| Less chance of context pollution | Binary-level integration tests | test_helpers_test.go |
| | 832-line format integration matrix | BFS cycle detection with full path |
| | DiscoverTickDir (walk-up) | Self-healing cache with detailed recovery |
| | Buffered JSONL writes | |
| | unwrapMutationError pattern | |

---

## Verdict

**Best implementation: V2 (implementation-v2)**

Reasons:
1. **SQL safety** — parameterized queries vs V1's string interpolation
2. **Most thorough tests** — 3.4:1 ratio, binary tests, exhaustive format matrix, corruption tests
3. **Better architecture** — storage sub-packages, DiscoverTickDir, cleaner separation
4. **Robust error handling** — unwrapMutationError, data-driven transitions

V2's weaknesses (`interface{}` in FormatStats, truncateTitle byte/rune bug, incomplete usage) are minor and easily fixable.

**V3 is a close second** with some better patterns (exported SQL constants, spec compliance tests, consolidated helpers) but has more practical issues (double file read, os.Stderr bypass, empty output for blocked_by).

**V1 is the weakest** despite being most compact. The SQL interpolation pattern is a real concern, the tests are thinnest, and the duplicated ready query is fragile.

---

## On the Core Question: Is the Agent-Based Approach Worth the Time Cost?

**Yes, but with caveats.**

The agent-based approach produces:
- **~2x more test code** with genuinely more edge cases covered (not just verbosity)
- **Better practices** in some areas (parameterized SQL, better separation)
- **Review-stage catches** that the sequential approach misses

The quality delta is real but **not dramatic** for the production code itself (~3.4k vs ~2.9k lines, similar architecture). The main value is in test thoroughness.

The V2-V3 difference suggests the approach has stabilized — both agent-based runs produce similar quality. V2 edges out V3, possibly because V2's agents had slightly better context/instructions, or because the review stage was more effective in V2's run.

If time cost is a major concern, a **hybrid approach** might be optimal: sequential implementation for the core code (fast), then agent-based review/testing pass (catches gaps). That would give V1's speed with V2's test quality.

---

## Per-Branch Analysis Reports

Detailed per-branch analysis reports are available:
- V1: `scratchpad/v1-analysis.md`
- V2: `scratchpad/v2-analysis.md`
- V3: `scratchpad/v3-analysis.md`
