---
status: complete
created: 2026-02-28
cycle: 2
phase: Plan Integrity Review
topic: cli-enhancements
---

# Review Tracking: cli-enhancements - Integrity

## Findings

### 1. Five tasks use summary test counts instead of named test lists

**Severity**: Minor
**Plan Reference**: Phase 3 / tick-7d56c4 (3-1), tick-f713ec (3-4), tick-56001c (3-5); Phase 4 / tick-6d5863 (4-3), tick-4b4e4b (4-4)
**Category**: Task Template Compliance
**Change Type**: update-task

**Details**:
The task template requires named test entries (e.g., `"it validates ref with comma is rejected"`). Five tasks instead use summary counts like "15 tests covering round-trip, omitempty, valid tags..." This is inconsistent with all other tasks in the plan which list individual test names. Named tests give the implementer explicit test-driven targets. The acceptance criteria in these tasks provide sufficient detail for implementation, so this is a style/consistency concern rather than a blocker.

**Current** (tick-7d56c4 Tests field):
```
Tests: 15 tests covering round-trip, omitempty, valid tags, regex rejections, normalization, dedup, max count, empty filtering.
```

**Proposed** (tick-7d56c4 Tests field):
```
Tests:
- "it marshals tags to JSON array"
- "it omits empty tags from JSON (omitempty)"
- "it unmarshals tags from JSON"
- "it validates valid kebab-case tag (frontend)"
- "it validates multi-segment kebab-case tag (ui-component)"
- "it rejects tag with double hyphens"
- "it rejects tag with leading hyphen"
- "it rejects tag with trailing hyphen"
- "it rejects tag with spaces"
- "it rejects tag exceeding 30 chars"
- "it accepts tag at exactly 30 chars"
- "it normalizes tag to trimmed lowercase"
- "it deduplicates tags preserving first-occurrence order"
- "it accepts 11 tags deduped to 10"
- "it rejects 11 unique tags"
- "it filters empty strings from tag list"
```

**Current** (tick-f713ec Tests field):
```
Tests: 14 tests covering create with/without tags, empty value, dedup, normalization, invalid format, max count, update set/clear, mutual exclusivity, idempotent clear, output verification.
```

**Proposed** (tick-f713ec Tests field):
```
Tests:
- "it creates a task with --tags ui,backend"
- "it creates a task without --tags (optional)"
- "it errors on create with empty --tags value"
- "it deduplicates tags on create"
- "it normalizes tag input to lowercase on create"
- "it rejects invalid kebab-case tag on create"
- "it rejects more than 10 unique tags on create"
- "it updates task tags with --tags api,frontend"
- "it clears task tags with --clear-tags"
- "it errors on update with --tags and --clear-tags together"
- "it errors on update with empty --tags value"
- "it succeeds with --clear-tags on task with no tags (idempotent)"
- "it persists tags to JSONL and shows in output"
- "it updates hasChanges correctly for --tags and --clear-tags"
```

**Current** (tick-56001c Tests field):
```
Tests: 14 tests covering single tag, AND, OR, composition, empty results, validation, normalization, combined filters.
```

**Proposed** (tick-56001c Tests field):
```
Tests:
- "it filters list by single tag"
- "it filters list by AND (comma-separated tags)"
- "it filters list by OR (multiple --tag flags)"
- "it filters list by AND/OR composition (--tag ui,backend --tag api)"
- "it returns empty list when no tasks match tag filter"
- "it rejects invalid kebab-case tag in filter"
- "it normalizes tag filter input to lowercase"
- "it filters ready tasks by tag"
- "it filters blocked tasks by tag"
- "it combines --tag with --status filter"
- "it combines --tag with --priority filter"
- "it combines --tag with --parent filter"
- "it combines --tag with --count flag"
- "it returns all tasks when --tag not specified"
```

**Current** (tick-6d5863 Tests field):
```
Tests: 9 tests covering create with/without refs, update replace/clear, mutual exclusivity, empty value, dedup, idempotent clear.
```

**Proposed** (tick-6d5863 Tests field):
```
Tests:
- "it creates a task with --refs gh-123,JIRA-456"
- "it creates a task without --refs (optional)"
- "it errors on create with empty --refs value"
- "it deduplicates refs on create"
- "it updates task refs with --refs new-ref"
- "it clears task refs with --clear-refs"
- "it errors on update with --refs and --clear-refs together"
- "it succeeds with --clear-refs on task with no refs (idempotent)"
- "it rejects invalid ref (contains whitespace) on create"
```

**Current** (tick-4b4e4b Tests field):
```
Tests: 7 tests covering each formatter with/without refs, max refs.
```

**Proposed** (tick-4b4e4b Tests field):
```
Tests:
- "it displays refs in pretty show output"
- "it omits refs section in pretty when no refs"
- "it displays refs in toon show output"
- "it displays refs in json show output"
- "it shows empty refs array in json when no refs"
- "it displays all 10 refs when task has maximum"
- "it does not show refs in list output"
```

**Resolution**: Fixed
**Notes**: All five tasks updated with individually named test lists.
