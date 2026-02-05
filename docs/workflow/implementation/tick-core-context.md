# Integration Context: Tick Core

## tick-core-1-1: Task model & ID generation

### Integration (executor)
- Core task model lives in `/Users/leeovery/Code/tick/internal/task/task.go` — Task struct, Status enum, and all validation/generation functions
- ID generation via `GenerateID(ExistsFunc)` — accepts a collision-check function, allowing storage layer to provide existence lookup
- Validation functions are separate from struct for flexibility: `ValidateTitle`, `ValidatePriority`, `ValidateBlockedBy`, `ValidateParent`
- `TrimTitle` is separate from `ValidateTitle` — call TrimTitle first, then ValidateTitle on trimmed result
- Timestamps use `time.RFC3339` format (ISO 8601 UTC) — `DefaultTimestamps()` returns both created and updated as same value

### Cohesion (reviewer)
- Error messages: lowercase, descriptive, no "Error:" prefix (e.g., `"title is required"`)
- Validation pattern: standalone functions returning `error` for each field type
- Type conventions: `Status` as named string type with constants, not iota
- Test structure: subtests with descriptive names matching "it does X" format
