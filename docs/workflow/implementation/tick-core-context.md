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
