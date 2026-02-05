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

## tick-core-1-2: JSONL storage with atomic writes

### Integration (executor)
- JSONL storage in `/Users/leeovery/Code/tick/internal/storage/jsonl.go` — `WriteJSONL(path, []task.Task)` and `ReadJSONL(path) ([]task.Task, error)`
- Atomic write pattern: temp file in same directory + fsync + rename — ensures crash safety
- Empty file returns empty slice (not nil); missing file returns `os.ErrNotExist`
- Relies on Task struct's `omitempty` JSON tags for optional field omission — no custom marshaling needed
- Field order in JSONL matches struct definition order (id, title, status, priority, optional fields, timestamps)

### Cohesion (reviewer)
- Storage package follows established pattern: internal/storage separate from internal/task
- Test naming convention: "it does X" format maintained
- Package doc comment present on jsonl.go
- Error handling: returns raw errors (acceptable for low-level storage layer)
