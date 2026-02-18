---
topic: doctor-validation
status: concluded
format: local-markdown
specification: ../specification/doctor-validation/specification.md
spec_commit: 29e62a8301438296c5b05db3e8f36c75ad5c20e1
created: 2026-01-30
updated: 2026-01-30
external_dependencies:
  - topic: tick-core
    description: Doctor validates against data schema, ID format, hierarchy rules, and cache structure
    state: resolved
    task_id: tick-core-1-1
  - topic: tick-core
    description: `tick rebuild` command referenced in cache staleness suggestion
    state: resolved
    task_id: tick-core-5-2
planning:
  phase: 6
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
| doctor-validation-1-1 | Check Interface & Diagnostic Runner | zero checks registered, all checks pass, all checks fail, mixed pass/fail | authored |
| doctor-validation-1-2 | Output Formatter & Exit Code Logic | zero issues (summary says no issues), single issue, multiple issues from one check, suggestion text present vs absent | authored |
| doctor-validation-1-3 | Cache Staleness Check | missing cache.db (report stale), missing tasks.jsonl, empty tasks.jsonl with matching hash, hash mismatch | authored |
| doctor-validation-1-4 | tick doctor Command Wiring | .tick directory not found, doctor never modifies data (read-only verification) | authored |

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

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| doctor-validation-2-1 | JSONL Syntax Check | empty file, blank/whitespace-only lines, all lines valid, all lines malformed, single malformed line among many valid, trailing newline producing empty last line, missing tasks.jsonl | authored |
| doctor-validation-2-2 | ID Format Check | empty ID field, missing ID field, uppercase hex chars, extra chars beyond 6 hex, wrong prefix, numeric-only random part, mixed valid and invalid IDs | authored |
| doctor-validation-2-3 | Duplicate ID Check | exact-case duplicates, mixed-case duplicates (tick-ABC123 vs tick-abc123), more than two duplicates of same ID, multiple distinct duplicate groups, no duplicates, single task | authored |
| doctor-validation-2-4 | Data Integrity Check Registration | all new checks pass alongside passing cache check, all new checks fail alongside passing cache check, mixed results across all four checks, empty tasks.jsonl | authored |

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

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| doctor-validation-3-1 | Orphaned Parent Reference Check | missing tasks.jsonl, empty file, all parents valid, multiple orphaned parents, parent field null/absent (valid root task), unparseable lines skipped | authored |
| doctor-validation-3-2 | Orphaned Dependency Reference Check | missing tasks.jsonl, empty file, single orphaned dep, multiple orphaned deps on same task, multiple orphaned deps across tasks, all deps valid, empty blocked_by array, unparseable lines skipped, blocked_by contains mix of valid and invalid refs | authored |
| doctor-validation-3-3 | Self-Referential Dependency Check | missing tasks.jsonl, empty file, task blocked by itself among other valid deps, multiple tasks each self-referential, task with only self-reference, unparseable lines skipped | authored |
| doctor-validation-3-4 | Dependency Cycle Detection Check | missing tasks.jsonl, empty file, simple 2-node cycle, 3+ node cycle, multiple independent cycles, chain that is not a cycle, self-reference not double-reported (handled by task 3-3), task with no dependencies, complex graph with both cycles and valid chains | authored |
| doctor-validation-3-5 | Child Blocked-By Parent Check | missing tasks.jsonl, empty file, direct child blocked by parent, child blocked by grandparent (not flagged -- only direct parent), multiple children blocked by same parent, child blocked by parent among other valid deps, task with parent but no blocked_by (valid) | authored |
| doctor-validation-3-6 | Parent Done With Open Children Warning | missing tasks.jsonl, empty file, parent done with one open child, parent done with multiple open children, parent done with all children done, parent done with cancelled children only, parent open with open children (not flagged), parent done with in_progress child, warnings-only produces exit code 0 | authored |
| doctor-validation-3-7 | Relationship Check Registration | all 10 checks pass (healthy store), mixed errors and warnings, warnings-only exit code 0, errors from relationship checks combine with earlier phase errors in summary count | authored |

### Phase 4: Analysis (cycle 1 — reduce duplication and improve architecture)
status: approved

**Goal**: Address findings from implementation analysis cycle 1.

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| doctor-validation-4-1 | Extract shared JSONL line iterator and parse tasks.jsonl once per doctor run | — | authored |
| doctor-validation-4-2 | Make tickDir an explicit parameter on the Check interface | — | authored |
| doctor-validation-4-3 | Extract fileNotFoundResult helper for repeated tasks.jsonl-not-found error | — | authored |

### Phase 5: Analysis (cycle 2 — derive relationships from shared parser, extract test helpers, use report methods)
status: approved

**Goal**: Address findings from implementation analysis cycle 2.

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| doctor-validation-5-1 | Derive ParseTaskRelationships from ScanJSONLines output | — | approved |
| doctor-validation-5-2 | Extract assertReadOnly test helper | — | approved |
| doctor-validation-5-3 | Use DiagnosticReport methods for issue count in FormatReport | — | approved |

### Phase 6: Analysis (cycle 3 — extract buildKnownIDs helper)
status: approved

**Goal**: Address findings from implementation analysis cycle 3.

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| doctor-validation-6-1 | Extract buildKnownIDs helper to eliminate 3-file duplication | — | approved |

---

## Log

| Date | Change |
|------|--------|
| 2026-01-30 | Created from specification |
