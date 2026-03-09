---
status: in-progress
created: 2026-03-09
cycle: 1
phase: Input Review
topic: unknown-flags-silently-ignored
---

# Review Tracking: unknown-flags-silently-ignored - Input Review

## Findings

### 1. Missing commands in blast radius and flag inventory: rebuild and migrate

**Source**: Investigation lines 105-106 ("Directly affected: All commands...") cross-referenced with actual dispatcher in app.go lines 87-117
**Category**: Enhancement to existing topic
**Affects**: Requirements (item 3), Command Flag Inventory (Per-Command Flags table)

**Details**:
The investigation lists "All commands" under blast radius but then enumerates a specific set that omits `rebuild` and `migrate`. The specification copied this enumeration. However, both commands exist in the dispatcher:

- `rebuild` (app.go:112) - routed through the normal dispatch switch, receives `subArgs`, accepts no flags. Currently its handler ignores args but the central validation should still cover it.
- `migrate` (app.go:55) - dispatched before format resolution (alongside `doctor`), has its own flags (`--from`, `--dry-run`, `--pending-only`), and its parser (`parseMigrateArgs` in migrate.go:47-69) also silently skips unknown flags (no default case in the switch).

`migrate` is particularly notable because it has the same bug pattern the spec aims to fix, but is dispatched through a different code path (before format config resolution, like `doctor`). If validation only happens in the normal dispatch switch, `migrate` would remain unfixed.

**Proposed Addition**:

**Resolution**: Approved
**Notes**: Added rebuild and migrate to Requirements item 3, Per-Command Flags table, and added Dispatch Paths subsection to Design.

---

### 2. migrate command uses --from=value equals-sign syntax

**Source**: migrate.go line 57: `case strings.HasPrefix(args[i], "--from=")`
**Category**: Enhancement to existing topic
**Affects**: Design (Flow/validation approach), Command Flag Inventory

**Details**:
The `migrate` command's parser supports `--from=value` syntax (equals-sign assignment) in addition to `--from value` (space-separated). This is the only command in the codebase using this pattern. A central validator that checks for unknown flags by matching args starting with `-` against a known flag set would need to handle the `--flag=value` form: `--from=beads` should not be rejected as an unknown flag just because it doesn't exactly match `--from`.

This is relevant regardless of whether `migrate` goes through the central validator or has its own validation, because the pattern could appear in future commands.

**Proposed Addition**:

**Resolution**: Pending
**Notes**: This may be a non-issue if `migrate` continues to be dispatched before the central validation point, but the spec should at minimum acknowledge the pattern exists so the implementer handles it correctly if `migrate` is brought under the central validator.
