---
status: complete
created: 2026-02-27
cycle: 1
phase: Input Review
topic: cli-enhancements
---

# Review Tracking: CLI Enhancements - Input Review

## Findings

### 1. Partial ID case sensitivity

**Source**: Discussion — "Currently IDs are resolved by exact match (case-insensitive via NormalizeID)"
**Category**: Enhancement to existing topic
**Affects**: Partial ID Matching

**Details**:
The discussion mentions existing ID resolution is case-insensitive via `NormalizeID`. The spec doesn't state whether partial prefix matching should also be case-insensitive. Since hex chars can be typed uppercase (`A3F`) or lowercase (`a3f`), this should be explicit.

**Proposed Addition**:
Add to the Resolution rules list: `- Case-insensitive: input normalized to lowercase before matching`

**Resolution**: Approved
**Notes**:

---

### 2. Note remove index validation

**Source**: Specification analysis
**Category**: Gap/Ambiguity
**Affects**: Notes

**Details**:
The spec says `tick note remove <id> <index>` removes by 1-based position but doesn't specify behavior for out-of-bounds indices (e.g., index 5 when only 3 notes exist, index 0, negative values).

**Proposed Addition**:
Add to the Notes Validation section: `- Note remove index must be >= 1 and <= number of existing notes; out-of-bounds errors`

**Resolution**: Approved
**Notes**:

---

### 3. Filter input normalization

**Source**: Discussion — "Case-insensitive input, trimmed, stored lowercase" (for type); "Input trimmed and lowercased" (for tags)
**Category**: Gap/Ambiguity
**Affects**: Tags (filtering), Task Types (filtering)

**Details**:
The spec specifies input normalization (trim, lowercase) for create/update but doesn't state whether filter flags (`--tag`, `--type` on list commands) also normalize input before matching. Without normalization, `--tag UI` wouldn't match stored `ui`.

**Proposed Addition**:
Add to Tags Filtering section: `- Filter input normalized (trimmed, lowercased) before matching`
Add to Task Types Filtering section: `- Filter input normalized (trimmed, lowercased) before matching`

**Resolution**: Approved
**Notes**:
