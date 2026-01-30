---
topic: doctor-validation
status: planning
format: local-markdown
specification: ../specification/doctor-validation.md
spec_commit: 29e62a8301438296c5b05db3e8f36c75ad5c20e1
created: 2026-01-30
updated: 2026-01-30
planning:
  phase: 1
  task: ~
---

# Plan: Doctor Validation

## Overview

**Goal**: Implement `tick doctor` — a diagnostic command that validates data store integrity and reports issues without modifying data.

**Done when**:
- All 9 error checks and 1 warning check run in a single invocation
- Human-readable output with ✓/✗ markers, details, and fix suggestions
- Exit code 0 when clean, 1 when errors found

**Key Decisions** (from specification):
- Report, don't fix — doctor diagnoses and suggests; never modifies data
- Human-focused — no structured output variants (no TOON/JSON)
- Run all checks — never stops early on first failure
- `tick rebuild` is out of scope — already planned in tick-core

## Phases

### Phase 1: Walking Skeleton — Doctor Framework & Cache Check
status: approved
approved_at: 2026-01-30

**Goal**: Prove the diagnostic runner end-to-end with a single real check (cache staleness). Establishes the check registration pattern, output formatting, exit codes, and suggestion mechanism.
**Why this order**: Foundation. Every subsequent phase adds checks to this framework. Must validate the runner pattern, output format, and exit code logic before adding more checks.

**Acceptance**:
- [ ] `tick doctor` runs all registered checks and reports results
- [ ] Cache staleness check detects hash mismatch between JSONL and SQLite
- [ ] Passing checks display ✓ with label
- [ ] Failing checks display ✗ with details and fix suggestion
- [ ] Cache staleness suggests "Run `tick rebuild` to refresh cache"
- [ ] Summary line shows total issue count
- [ ] Exit code 0 when all checks pass, exit code 1 when any error found
- [ ] Doctor never modifies data

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| doctor-validation-1-1 | Check Interface & Diagnostic Runner | zero checks registered, all checks pass, all checks fail, mixed pass/fail | pending |
| doctor-validation-1-2 | Output Formatter & Exit Code Logic | zero issues (summary says no issues), single issue, multiple issues from one check, suggestion text present vs absent | pending |
| doctor-validation-1-3 | Cache Staleness Check | missing cache.db (report stale), missing tasks.jsonl, empty tasks.jsonl with matching hash, hash mismatch | pending |
| doctor-validation-1-4 | tick doctor Command Wiring | .tick directory not found, doctor never modifies data (read-only verification) | pending |

### Phase 2: Data Integrity Checks
status: approved
approved_at: 2026-01-30

**Goal**: Validate JSONL file integrity — syntax, ID uniqueness, and ID format. These checks operate on raw file content without relationship semantics.
**Why this order**: Builds on Phase 1's framework. Data-level checks are simpler than relationship checks and validate preconditions (parseable data, valid IDs) that relationship checks implicitly depend on.

**Acceptance**:
- [ ] JSONL syntax check detects and reports each malformed line with details
- [ ] Duplicate ID check detects case-insensitive duplicates (tick-ABC123 = tick-abc123)
- [ ] ID format check detects IDs not matching prefix + 6 hex chars format
- [ ] Multiple errors of the same type each reported individually with specifics
- [ ] All checks run even if earlier checks find errors
- [ ] Errors from all checks contribute to summary count and exit code

### Phase 3: Relationship & Hierarchy Checks
status: approved
approved_at: 2026-01-30

**Goal**: Validate all task relationship constraints — orphaned references, dependency integrity, and hierarchy warnings. Completes the full doctor check suite.
**Why this order**: Requires the framework (Phase 1) and benefits from data integrity checks existing (Phase 2). Relationship checks are the most complex validation logic and form a cohesive group.

**Acceptance**:
- [ ] Orphaned parent reference check detects tasks referencing non-existent parents
- [ ] Orphaned dependency reference check detects tasks depending on non-existent tasks
- [ ] Self-referential dependency check detects tasks depending on themselves
- [ ] Dependency cycle check detects circular chains (A→B→C→A)
- [ ] Child blocked_by parent check detects deadlock conditions
- [ ] Warning: parent done with open children reported with ⚠ or appropriate marker
- [ ] Warnings do not affect exit code (exit 0 if only warnings, no errors)
- [ ] All 10 checks (9 errors + 1 warning) run in a single `tick doctor` invocation

---

## External Dependencies

- tick-core: Doctor validates against data schema, ID format, hierarchy rules, and cache structure → tick-core plan (resolved)
- tick-core: `tick rebuild` command referenced in cache staleness suggestion → tick-core-5-2 (resolved)

## Log

| Date | Change |
|------|--------|
| 2026-01-30 | Created from specification |
