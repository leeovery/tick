---
status: in-progress
created: 2026-02-09
phase: Traceability Review
topic: tick-core
---

# Review Tracking: tick-core - Traceability Review

## Findings

### 1. Phase 3 acceptance criteria missing --parent flag

**Type**: Incomplete coverage
**Spec Reference**: List Command Options — `--parent <id>  Scope to descendants of this task (recursive)`; Parent Scoping section
**Plan Reference**: Phase 3 acceptance criteria

**Details**:
Phase 3 acceptance criteria lists `tick list` supports `--ready, --blocked, --status, --priority` filters but does not mention `--parent`. The spec now includes `--parent` as a list command option and has a full Parent Scoping section. The phase acceptance should reflect this.

**Proposed Fix**:
Add acceptance criterion to Phase 3.

**Resolution**: Pending
**Notes**:
