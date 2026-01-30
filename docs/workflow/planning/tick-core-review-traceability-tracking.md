---
status: in-progress
created: 2026-01-30
phase: Traceability Review
topic: tick-core
---

# Review Tracking: tick-core - Traceability Review

## Findings

### 1. List ordering not specified in spec

**Type**: Incomplete coverage (minor inference)
**Spec Reference**: Core Values: "Deterministic queries - Same input, same output, always"
**Plan Reference**: tick-core-1-7, tick-core-3-3, tick-core-3-4

**Details**:
The plan specifies `priority ASC, created ASC` as the list ordering. The spec never defines an explicit sort order for `tick list`, `tick ready`, or `tick blocked`. However, the spec's core value of "Deterministic queries" requires _some_ stable ordering. The chosen order (highest priority first, oldest first) is reasonable but not spec-derived.

**Proposed Fix**:

**Resolution**: Pending
**Notes**:

---

### 2. `.tick/` directory discovery walk-up

**Type**: Incomplete coverage (reasonable inference)
**Spec Reference**: Init Command: "Error: Not a tick project (no .tick directory found)"
**Plan Reference**: tick-core-1-5 (Implementation + Edge Cases)

**Details**:
tick-core-1-5 specifies walking up from cwd to filesystem root to find `.tick/`. The spec doesn't describe this behavior — it only says the error message when no `.tick/` is found. Walking up is standard CLI behavior (git, npm, cargo all do this) and is necessary for subdirectory operation, but the specific mechanism isn't in the spec.

**Proposed Fix**:

**Resolution**: Pending
**Notes**:

---

### 3. Malformed JSONL line handling

**Type**: Incomplete coverage (implementation gap)
**Spec Reference**: N/A — spec silent on malformed JSONL
**Plan Reference**: tick-core-5-2 Edge Cases: "Malformed JSONL lines: skip with warning"

**Details**:
tick-core-5-2 mentions "skip with warning (same as normal cache build)" for malformed JSONL lines, implying the normal JSONL reader also skips malformed lines. However, tick-core-1-2 (the JSONL reader) doesn't specify this behavior, and the spec doesn't address malformed JSONL at all. The plan should be consistent — either both tasks mention it or neither does.

**Proposed Fix**:

**Resolution**: Fixed
**Notes**: Updated tick-core-5-2 edge case to remove unsupported claim

---

### 4. No subcommand behavior

**Type**: Incomplete coverage (minor)
**Spec Reference**: N/A — spec silent on no-subcommand case
**Plan Reference**: tick-core-1-5 Edge Cases: "No subcommand: print basic usage with exit code 0"

**Details**:
tick-core-1-5 specifies printing basic usage when no subcommand is given. The spec doesn't mention this. Standard CLI behavior, but strictly not spec-derived.

**Proposed Fix**:

**Resolution**: Pending
**Notes**:

---

### 5. Doctor-related spec content deferred correctly

**Type**: Verification (not a finding)
**Spec Reference**: Edge Cases (orphaned children, parent done before children) — lines 425-431
**Plan Reference**: Phase 5 goal note: "Doctor/validation deferred to separate doctor-validation specification"

**Details**:
The spec mentions `tick doctor` flags for orphaned children and parent-done-before-children. These are correctly excluded from this plan since doctor has its own concluded specification. Verified — no gap.

**Proposed Fix**:

**Resolution**: N/A — correct as-is
**Notes**: User explicitly approved deferral
