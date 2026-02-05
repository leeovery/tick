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

## tick-core-1-3: SQLite cache with freshness detection

### Integration (executor)
- SQLite cache in `/Users/leeovery/Code/tick/internal/storage/cache.go` — `Cache` struct wraps `*sql.DB`, path stored for corruption recovery
- `NewCache(path)` creates cache.db with schema; `EnsureFresh(path, tasks, jsonlContent)` is the gatekeeper to call on every operation
- Hash stored in metadata table under key `jsonl_hash` — use SHA256 of raw JSONL bytes
- Dependencies normalized to `dependencies` table with composite PK `(task_id, blocked_by)` — query this table for blocking relationships instead of the JSONL array
- Rebuild is transactional — all-or-nothing via `tx.Begin()`/`tx.Commit()`; rollback automatic on error via `defer tx.Rollback()`

### Cohesion (reviewer)
- Package structure follows established pattern: storage package contains both JSONL and SQLite cache
- Error handling follows project convention: returns raw errors without wrapping
- Test naming follows "it does X" format consistently
- Cache struct mirrors established patterns: constructor returns pointer with error, Close method for cleanup

## tick-core-1-4: Storage engine with file locking

### Integration (executor)
- Created `Store` in `/Users/leeovery/Code/tick/internal/storage/store.go` — use for all task read/write operations; composes JSONL and SQLite cache
- `Store.Mutate(func([]task.Task) ([]task.Task, error))` — exclusive lock, receives current tasks, returns modified tasks; handles atomic write flow
- `Store.Query(func(*sql.DB) error)` — shared lock, provides direct SQLite access for queries; automatically ensures cache freshness
- Lock timeout is 5 seconds; error message: "could not acquire lock on .tick/lock - another process may be using tick"
- JSONL-first principle: SQLite failures during mutation are logged to stderr but return success (next read self-heals)

### Cohesion (reviewer)
- Store constructor pattern (`NewStore(path) (*Store, error)`) matches Cache constructor pattern from tick-core-1-3
- Error messages follow project convention: lowercase, descriptive, no "Error:" prefix
- Test naming follows "it does X" format consistently with prior tasks
- Lock path `.tick/lock` and cache path `.tick/cache.db` follow directory structure from spec

## tick-core-1-5: CLI framework & tick init

### Integration (executor)
- CLI entry point at `cmd/tick/main.go` — use `go build ./cmd/tick` to build the binary
- CLI application in `internal/cli/cli.go` with testable `App` struct — inject `Stdout`, `Stderr`, `Cwd` for testing
- `DiscoverTickDir(startDir string) (string, error)` in `internal/cli/cli.go` — walks up from cwd to find `.tick/`, returns error "not a tick project (no .tick directory found)" if not found
- Global flags in `GlobalFlags` struct: `Quiet`, `Verbose`, `OutputFormat` — use `ParseGlobalFlags(args)` to extract
- `IsTTY(io.Writer) bool` detects terminal for output format auto-selection — non-TTY defaults to TOON, TTY defaults to pretty

### Cohesion (reviewer)
- CLI layer uses "Error: " prefix for user-facing error messages (distinct from internal error messages which are lowercase without prefix)
- App struct pattern with injectable Stdout/Stderr/Cwd enables testability
- GlobalFlags struct with OutputFormat string allows flag-based format override or TTY auto-detection
- Test naming consistently follows "it does X" format established in prior tasks

## tick-core-1-6: tick create command

### Integration (executor)
- Created `/Users/leeovery/Code/tick/internal/cli/create.go` with `runCreate()` method, `parseCreateArgs()` for flag parsing, and `printTaskDetails()` for output formatting
- Flag parsing pattern: positional arg for title, `--flag value` style for options
- `--blocks` flag is the inverse of `--blocked-by`: modifies target tasks' `blocked_by` arrays and refreshes their `updated` timestamps atomically
- `normalizeIDs()` helper in create.go handles slice normalization; use `task.NormalizeID()` for single IDs
- Error messages follow CLI layer pattern: "Error: <message>\n" to stderr with exit code 1

### Cohesion (reviewer)
- Error message convention maintained: user-facing errors use "Error: " prefix written to stderr
- Store.Mutate pattern correctly used for atomic writes with SQLite cache update
- ID normalization uses task.NormalizeID() consistently across blocked_by, blocks, and parent
- Test naming follows established "it does X" format consistently with prior tasks

## tick-core-1-7: tick list & tick show commands

### Integration (executor)
- `runList()` in `/Users/leeovery/Code/tick/internal/cli/list.go` — queries SQLite via `Store.Query()`, ordered by priority ASC then created ASC
- `runShow()` in `/Users/leeovery/Code/tick/internal/cli/show.go` — fetches task + relationships (blocked_by from dependencies table, children via parent field)
- Both commands use `Store.Query()` for shared-lock read flow — ensures cache freshness automatically
- Shared test helpers extracted to `/Users/leeovery/Code/tick/internal/cli/test_helpers_test.go` following Go convention

### Cohesion (reviewer)
- Test helpers extracted to test_helpers_test.go following Go convention of _test.go files sharing package scope
- Consistent error message format "Error: ..." followed by details
- Commands follow same pattern: discover tick dir, open store, query, format output
- strconv.Itoa pattern used for priority conversion in test data setup

## tick-core-2-1: Status transition validation logic

### Integration (executor)
- `Transition(task *Task, command string) (TransitionResult, error)` in `/Users/leeovery/Code/tick/internal/task/transition.go` — pure domain logic for status transitions
- `TransitionResult{OldStatus, NewStatus}` — returns both statuses for CLI output formatting (e.g., "tick-a3f2b7: open -> in_progress")
- Timestamps handled internally: `updated` refreshed on every valid transition, `closed` set on done/cancel, cleared on reopen
- Task pointer mutated in place on success; left unmodified on error

### Cohesion (reviewer)
- TransitionResult struct follows project pattern of returning structured data with named fields
- Error message format "cannot %s task %s - status is '%s'" maintains lowercase convention
- time.RFC3339 usage consistent with DefaultTimestamps() in task.go
- Test structure (helper functions at top, valid cases, invalid cases, timestamp tests) provides good organizational pattern

## tick-core-2-2: tick start, done, cancel, reopen commands

### Integration (executor)
- `runTransition(command, args)` in `/Users/leeovery/Code/tick/internal/cli/transition.go` — single handler for all four transition commands
- Output format: `{id}: {old_status} → {new_status}\n` with unicode arrow (→), uses normalized lowercase ID
- Commands registered in cli.go switch: `case "start", "done", "cancel", "reopen":` routes to common handler
- Uses existing patterns: `DiscoverTickDir()`, `Store.Mutate()`, `task.NormalizeID()`, `task.Transition()`

### Cohesion (reviewer)
- Pattern established: transition commands share single handler (runTransition) with command parameter
- Error message convention maintained: "Error: task 'tick-xyz' not found" format
- Store.Mutate correctly returns modified slice after in-place mutation via pointer
- Test helper setupTaskFull enables testing tasks in various states

## tick-core-2-3: tick update command

### Integration (executor)
- `runUpdate()` in `/Users/leeovery/Code/tick/internal/cli/update.go` — follows same pattern as runCreate and runTransition
- `UpdateFlags` struct uses pointers for optional fields (`*string`, `*int`) to distinguish "not provided" from "empty value"
- `--description ""` clears description; `--parent ""` clears parent
- `--blocks` adds this task to targets' `blocked_by`, refreshes their `updated` timestamps

### Cohesion (reviewer)
- UpdateFlags uses pointer types (*string, *int) for optional field detection — good pattern for update commands
- --blocks flag pattern established: modifies target tasks' blocked_by arrays atomically
- Reuses existing helpers: normalizeIDs(), printTaskDetails(), DiscoverTickDir(), Store.Mutate()
- Test helper reuse confirmed: setupTaskFull() enables testing tasks with specific initial states

## tick-core-3-1: Dependency validation — cycle detection & child-blocked-by-parent

### Integration (executor)
- `ValidateDependency(tasks []Task, taskID, newBlockedByID string) error` in `/Users/leeovery/Code/tick/internal/task/dependency.go` — validates single dependency
- `ValidateDependencies(tasks, taskID, blockedByIDs)` for batch validation — fails on first error
- Cycle error format: `cannot add dependency - creates cycle: tick-a → tick-b → tick-a` (unicode arrow)
- Child-blocked-by-parent error includes explanatory line about leaf-only ready rule

### Cohesion (reviewer)
- Uses established NormalizeID helper for consistent ID handling
- Error message style matches existing codebase (lowercase, fmt.Errorf pattern)
- Pure function design aligns with other validation functions in the task package
- BFS cycle detection correctly finds shortest path

## tick-core-3-2: tick dep add & tick dep rm commands

### Integration (executor)
- `/Users/leeovery/Code/tick/internal/cli/dep.go` with `runDep()`, `runDepAdd()`, `runDepRm()` — follows established CLI patterns
- `dep add` validates both task_id and blocked_by_id exist; `dep rm` only validates task_id (allows removing stale refs)
- Uses `task.ValidateDependency()` from tick-core-3-1 for cycle detection and child-blocked-by-parent validation
- Output: "Dependency added: {task_id} blocked by {blocked_by_id}" and "Dependency removed: {task_id} no longer blocked by {blocked_by_id}"

### Cohesion (reviewer)
- Error message pattern maintained: "Error: <lowercase message>\n" to stderr with exit code 1
- Store.Mutate pattern correctly used for atomic writes
- ID normalization via task.NormalizeID() consistently applied to both positional arguments
- Test naming follows established "it does X" format
