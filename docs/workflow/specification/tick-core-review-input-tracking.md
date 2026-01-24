---
status: in-progress
created: 2026-01-24
phase: Input Review
topic: tick-core
---

# Review Tracking: tick-core - Input Review

## Findings

### 1. ID Prefix Customization Discrepancy

**Source**: id-format-implementation.md - "What's settled" section
**Category**: Potential conflict between discussions
**Affects**: ID Generation section

**Details**:
The id-format-implementation discussion states in its "What's settled" section: "Prefix customizable at `tick init` (default: `tick`)". However, the spec says "Prefix: `tick` (hardcoded, no configuration)".

This appears to be intentionally resolved by the project-fundamentals non-goals decision: "Config file | YAGNI - hardcoded defaults are fine". The spec's "hardcoded" approach aligns with the overall project philosophy.

**Proposed Addition**: None - spec is correct. The discussion predates the config deferral decision.

**Resolution**: Skipped
**Notes**: Spec is correct. Discussion predates config-deferral decision.

---

### 2. Cache File Location Discrepancy

**Source**: freshness-dual-write.md - Context section
**Category**: Potential conflict between discussions
**Affects**: File Locations & Formats section

**Details**:
The freshness-dual-write discussion mentions `.cache/tick.db` as the SQLite location. The spec uses `.tick/cache.db`. These are different paths.

The spec's approach (`.tick/cache.db`) is cleaner - all tick files live under `.tick/`.

**Proposed Addition**: None - spec appears to have the correct/intended path.

**Resolution**: Skipped
**Notes**: Spec is correct. `.tick/` container is cleaner, matches "clean uninstall" criterion.

---

### 3. `tick stats` Command Output Not Specified

**Source**: cli-command-structure-ux.md - Final command reference
**Category**: Gap/Ambiguity
**Affects**: CLI Commands section

**Details**:
The `tick stats` command is listed in the command reference but has no specification of what it outputs. What statistics does it show? What format?

**Proposed Addition**: Added stats output specification with TOON and human-readable examples

**Resolution**: Approved
**Notes**: User decided to keep stats in v1. Added output examples for both formats showing: total, counts by status, ready/blocked counts, counts by priority.

---

### 4. Error Format for Non-TTY Unclear

**Source**: cli-command-structure-ux.md Q4, toon-output-format.md Q4
**Category**: Gap/Ambiguity
**Affects**: Error Handling section (under CLI Commands)

**Details**:
The cli-command-structure-ux discussion shows TTY-aware error formatting:
- Human at terminal: Friendly message
- Agent via pipe: Structured (TOON) with error code, message, context

But toon-output-format decided: "Plain text errors to stderr. Standard Unix convention."

The spec says: "All errors go to stderr. Plain text format (human-readable)."

These seem to conflict. Should errors be TOON-formatted for agents, or always plain text?

**Proposed Addition**: None - spec is correct

**Resolution**: Skipped
**Notes**: Confirmed plain text errors always. The cli-command-structure-ux TOON error suggestion was superseded by toon-output-format decision.

---

### 5. `tick rebuild` Command Detail

**Source**: freshness-dual-write.md, cli-command-structure-ux.md
**Category**: Enhancement to existing topic
**Affects**: CLI Commands section

**Details**:
The command reference lists `tick rebuild` and the spec mentions it "forces rebuild", but there's no detail on:
- What exactly happens when you run it?
- Does it have any flags?
- What output does it produce?

**Proposed Addition**: Added Rebuild Command section with description and use cases

**Resolution**: Approved
**Notes**: Added after Verbosity Flags section.

---

### 6. `tick doctor` Command Detail

**Source**: hierarchy-dependency-model.md - Edge cases section
**Category**: Enhancement to existing topic
**Affects**: CLI Commands section

**Details**:
The hierarchy discussion specifies what `tick doctor` should check:
- Orphaned children: "tick-child references non-existent parent tick-deleted"
- Parent done before children: optionally warn "tick-epic is done but has open children"

The spec only says `tick doctor | Run diagnostics and validation` without listing what it validates.

Note: There's a separate `doctor-command-validation` discussion that's NOT a source for this spec.

**Proposed Addition**: None - defer to doctor-command-validation discussion

**Resolution**: Skipped
**Notes**: Separate discussion exists for doctor command. Current command reference ("Run diagnostics and validation") is sufficient for this spec.

---

### 7. Suggested Error Message Examples

**Source**: hierarchy-dependency-model.md - Decision section
**Category**: Enhancement to existing topic
**Affects**: Hierarchy & Dependency section or Error Handling

**Details**:
The hierarchy discussion provides specific error message examples for invalid dependencies:
```
Error: Cannot add dependency - tick-child cannot be blocked by its parent tick-epic
       (would create unworkable task due to leaf-only ready rule)

Error: Cannot add dependency - creates cycle: tick-a → tick-b → tick-c → tick-a
```

The spec has the validation rules table but not these specific message formats.

**Proposed Addition**: None - already present

**Resolution**: Skipped
**Notes**: Error messages already exist in the spec at lines 349-355 in the Dependency Validation Rules section.

---

### 8. Collision Error Message Suggestion

**Source**: id-format-implementation.md - Collision handling section
**Category**: Enhancement to existing topic
**Affects**: ID Generation section

**Details**:
The discussion suggests a specific error message for collision exhaustion:
"failed to generate unique ID after 5 attempts - consider archiving completed tasks"

The spec says "If still colliding after 5 retries: return error" but doesn't specify the message.

Note: Archive is a non-goal, so the message may need adjustment.

**Proposed Addition**: Added collision error message (without archive reference)

**Resolution**: Approved
**Notes**: Added after collision handling bullet points.

---

### 9. SQLite Metadata Table for Hash Storage

**Source**: freshness-dual-write.md
**Category**: Enhancement to existing topic
**Affects**: SQLite Schema section

**Details**:
The spec includes the metadata table in the schema:
```sql
CREATE TABLE metadata (
  key TEXT PRIMARY KEY,
  value TEXT
);
```

And mentions: "metadata table stores the JSONL content hash for freshness detection"

But doesn't explicitly state the key used. The freshness discussion says: "key: `jsonl_hash`"

**Proposed Addition**: Added key name to metadata table description

**Resolution**: Approved
**Notes**: Updated inline description to include "(key: `jsonl_hash`)"

**Resolution**: Pending
**Notes**:
