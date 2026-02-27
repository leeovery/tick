---
topic: cli-enhancements
status: in-progress
type: feature
date: 2026-02-27
review_cycle: 0
finding_gate_mode: gated
sources:
  - name: cli-enhancements
    status: pending
---

# Specification: CLI Enhancements

## Specification

### Partial ID Matching

Allow users to reference tasks by a prefix of the hex portion instead of the full `tick-XXXXXX` ID.

**Resolution rules:**
- Both `tick-a3f` and `a3f` accepted — strip `tick-` prefix if present before matching
- Exact full-ID match takes priority: if input matches a complete ID (10 chars: `tick-` + 6 hex), return immediately without checking for prefix collisions
- Prefix matching only for inputs shorter than a full ID
- Minimum 3 hex chars required for prefix matching (prevents overly broad matches)
- Ambiguity (2+ matches): error listing the matching IDs
- Zero matches: "not found" error

**Implementation location:**
- Storage layer — `ResolveID(prefix)` method querying `WHERE id LIKE 'tick-{prefix}%'`
- Centralized: all commands resolve first, then proceed with the full ID
- Applies everywhere an ID is accepted: positional args, `--parent`, `--blocked-by`, `--blocks`

---

## Working Notes
