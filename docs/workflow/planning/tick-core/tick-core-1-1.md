---
id: tick-core-1-1
phase: 1
status: pending
created: 2026-01-30
---

# Task model & ID generation

## Goal

Tick needs a core data structure representing tasks and a deterministic ID format. Without this, no other component can operate. Define a Go struct with all 10 task fields, field validation logic, and a `tick-{6 hex}` ID generator using `crypto/rand` with collision retry.

## Implementation

- Define `Task` struct with fields: `id`, `title`, `status`, `priority`, `description`, `blocked_by`, `parent`, `created`, `updated`, `closed`
- Implement `Status` type as string enum with constants: `open`, `in_progress`, `done`, `cancelled`
- Implement ID generation: 3 random bytes from `crypto/rand` → 6 lowercase hex chars → prefix with `tick-`
- ID collision retry: accept a function to check existence, retry up to 5 times, error after that
- Normalize IDs to lowercase on input (case-insensitive matching)
- Validate title: required, non-empty, max 500 chars, no newlines, trim whitespace
- Validate priority: integer 0-4, reject out of range
- Validate `blocked_by`: no self-references (cycle detection is Phase 3)
- Validate `parent`: no self-references
- All timestamps use ISO 8601 UTC format (`YYYY-MM-DDTHH:MM:SSZ`)

## Tests

- `"it generates IDs matching tick-{6 hex} pattern"`
- `"it retries on collision up to 5 times"`
- `"it errors after 5 collision retries"`
- `"it normalizes IDs to lowercase"`
- `"it rejects empty title"`
- `"it rejects title exceeding 500 characters"`
- `"it rejects title with newlines"`
- `"it trims whitespace from title"`
- `"it rejects priority outside 0-4"`
- `"it rejects self-reference in blocked_by"`
- `"it rejects self-reference in parent"`
- `"it sets default priority to 2 when not specified"`
- `"it sets created and updated timestamps to current UTC time"`

## Edge Cases

- ID collision retry (5 attempts) then error with message: "Failed to generate unique ID after 5 attempts - task list may be too large"
- Title length limit (500 chars) — reject with error
- Title whitespace trimming — leading/trailing whitespace removed silently
- Title newlines — rejected (single line only)
- Priority range validation (0-4) — reject out of range
- Self-reference in blocked_by or parent — rejected

## Acceptance Criteria

- [ ] Task struct has all 10 fields with correct Go types
- [ ] ID format matches `tick-{6 hex chars}` pattern
- [ ] IDs are generated using `crypto/rand`
- [ ] Collision retry works up to 5 times then errors
- [ ] Input IDs are normalized to lowercase
- [ ] Title validation enforces non-empty, max 500 chars, no newlines, trims whitespace
- [ ] Priority validation rejects values outside 0-4
- [ ] Self-references in `blocked_by` and `parent` are rejected
- [ ] Timestamps are ISO 8601 UTC

## Context

Task schema has 10 fields. Optional fields (`description`, `blocked_by`, `parent`, `closed`) should use Go zero values/nil. Status enum: `open`, `in_progress`, `done`, `cancelled`. Priority is integer 0 (highest) to 4 (lowest), default 2.

Specification reference: `docs/workflow/specification/tick-core.md` (for ambiguity resolution)
