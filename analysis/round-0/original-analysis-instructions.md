# Branch Comparison Analysis Instructions

Instructions for running a comparative code quality analysis across implementation branches of the same plan.

---

## Objective

Compare multiple implementations of the same plan (same specification, same tasks) built using different workflow approaches. Determine which produces better code across architecture, edge cases, Go standards, test quality, and maintainability.

---

## Scope

- **Repository:** This repository (`tick`)
- **Plan:** `docs/workflow/planning/tick-core.md` and `docs/workflow/planning/tick-core/` (task files)
- **Branches to compare:** Each branch implements the same plan from a clean main branch
- **Exclude:** The `.claw/` directory in all branches (contains workflow system files, not project code)
- **Include:** All Go code (`*.go` files) — both implementation and test files

### Current Branches

| Branch | Approach | Description |
|---|---|---|
| `implementation` | V1 — Sequential single-agent | Main Claude session executes all tasks directly in one continuous context |
| `implementation-take-two` | V2 — Agent-based with review | Orchestrator dispatches fresh executor + reviewer sub-agents per task |

Future branches (e.g., `implementation-take-three`) should be added to this table and included in the analysis.

---

## Setup

Both branches must be accessible simultaneously. Options:

1. **Git worktrees** (recommended): `git worktree add <path> <branch>` for each branch
2. **Temporary clones**: Copy the repository to a temp directory and check out a different branch
3. **Sequential with intermediate files**: Analyse one branch, write findings to a file, check out the next, then compare

Parallel access via worktrees enables concurrent analysis and is strongly preferred.

---

## Analysis Process

### Phase 1: Structural Overview

For each branch:
1. List all Go files (`*.go`, excluding `.claw/`)
2. Count lines by package (implementation vs test)
3. Note package structure differences (flat vs sub-packages, file naming)
4. Run `go vet ./...` and `go test ./...` — record pass/fail counts
5. Count total test cases via `go test ./... -v 2>&1 | grep -c "=== RUN"`

Produce a metrics table comparing all branches.

### Phase 2: Layer-by-Layer Deep Comparison

Compare each architectural layer across all branches. Use parallel agents where possible — one per layer, reading from all worktrees simultaneously.

| Layer | Path | What to Assess |
|---|---|---|
| Task Model | `internal/task/` | Domain modeling, validation, transitions, dependency/cycle detection, edge cases |
| Storage | `internal/storage/` | JSONL atomicity, SQLite cache, freshness detection, locking, corruption handling |
| CLI Commands | `internal/cli/` (commands) | Flag parsing, validation, SQL queries, error handling, command wiring |
| Formatters | `internal/cli/` (formatters) | Interface design, TTY detection, TOON/pretty/JSON output, format selection |
| Entry Point | `cmd/tick/` | Bootstrapping, dependency wiring, error propagation |

For each layer, produce a structured comparison covering:
- **Architecture & organization**: Package structure, file decomposition, separation of concerns
- **Correctness**: Bugs, edge cases handled, validation thoroughness
- **Go idioms**: Error handling, naming, interfaces, constructors, resource cleanup
- **Test quality**: Coverage, table-driven tests, edge cases, test helpers, assertion style
- **Code examples**: Specific snippets where one implementation is clearly better, with line references

Each layer comparison should end with a clear verdict and score.

### Phase 3: Cross-Cutting Analysis

After layer comparisons, assess cross-cutting concerns:
- **DRYness**: Duplicated code across files/packages
- **Convention consistency**: Error message style, naming patterns, file organization
- **Interface design**: Type safety, domain coupling, extensibility
- **SQL safety**: Parameterized queries vs string interpolation
- **Resource lifecycle**: Connection management, cleanup, leak prevention

### Phase 4: Root Cause Attribution

For each advantage found in any branch, categorize the root cause:

| Category | Description |
|---|---|
| **A. Context Continuity** | Better because the agent had visibility of previously-written code |
| **B. Holistic Design** | Better because the agent could make cross-cutting architectural decisions |
| **C. Incremental Refinement** | Better because conventions evolved consistently over time |
| **D. Coincidental** | Better but not attributable to the workflow approach |

Produce a distribution table showing what percentage of advantages trace to each root cause, per branch.

### Phase 5: Synthesis

Produce a final verdict including:
1. **Overall winner** with justification
2. **Hard numbers table** (lines, tests, ratios)
3. **Layer-by-layer scorecard**
4. **Critical advantages** per branch (things that affect correctness)
5. **Genuine advantages** per branch (things that are better but fixable)
6. **Root cause analysis** of each branch's advantages
7. **Recommendations** for workflow improvements based on findings

---

## Output

Write the full analysis to `docs/implementation-comparison-v1-v2-v3.md` (or appropriate filename reflecting which branches were compared). Include all tables, code examples, and specific line references.

If a previous comparison document exists (e.g., `docs/implementation-comparison-v1-v2.md`), the new analysis should be standalone — incorporating previous findings where relevant but providing fresh analysis of all branches.

---

## Notes

- The plan files should be identical across branches. Verify this before starting.
- Focus on code quality differences, not on the workflow system itself (the `.claw/` directory).
- Be specific — cite file paths and line numbers for every claim.
- Use parallel agents for layer analysis to reduce wall-clock time.
- Both `go vet` and `go test` must be run on each branch to verify baseline correctness.
