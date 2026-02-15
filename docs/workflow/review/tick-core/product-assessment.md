SCOPE: single-plan
PLANS_REVIEWED: tick-core (Phases 1-5 implemented; Phases 6-8 pending)

ROBUSTNESS:

- **`log.Printf` bypasses injected writers** (`/Users/leeovery/Code/tick/internal/storage/store.go:167,273,288`). Three `log.Printf("warning: ...")` calls in Store use Go's default global logger (which writes to stderr with a timestamp prefix) rather than the app's injected `Stderr` writer. This means (a) cache corruption warnings are not testable via the App's captured output, (b) the timestamp prefix is inconsistent with all other error messages, and (c) in library/embedded usage the caller has no control over where these warnings go. These should route through the Store's `verboseLog` callback or a dedicated warning callback.

- **`truncateTitle` uses byte length, not rune count** (`/Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:162-167`). `len(title)` counts bytes, but `ValidateTitle` validates using `utf8.RuneCountInString` (`/Users/leeovery/Code/tick/internal/task/task.go:160`). A title of 50 multi-byte characters (e.g., CJK) could be 150 bytes, causing truncation at byte 47 -- potentially splitting a multi-byte rune in half and producing invalid UTF-8. The fix is to use `utf8.RuneCountInString` and `[]rune` slicing.

- **`dep add` and `dep rm` compare IDs without normalization** (`/Users/leeovery/Code/tick/internal/cli/dep.go:75,87,98,154`). While `parseDepArgs` normalizes both input IDs to lowercase, the comparisons inside `RunDepAdd` (finding task by ID at line 75, finding blocker at line 87, checking duplicate at line 98) and `RunDepRm` (finding task at line 148, checking dep at line 154) use direct `==` comparison against `tasks[i].ID`. If IDs stored in JSONL happen to have mixed case (unlikely given ID generation, but possible through manual editing), these lookups would fail. Other commands (create, update) consistently use `task.NormalizeID()` for comparison. This is a minor inconsistency but reduces robustness against hand-edited JSONL.

- **No maximum file size / task count guard**. Every `Mutate` and `Query` call reads the entire `tasks.jsonl` into memory (`os.ReadFile` at `/Users/leeovery/Code/tick/internal/storage/store.go:248`), parses all tasks, and on writes serializes the entire file back. For the intended use case (project-level task tracking, dozens to low hundreds of tasks) this is fine. But there is no guard or warning if the file grows unexpectedly large. Given the 6-hex-char ID space (16.7M possible IDs, collision retries at 5), this is an extremely low-priority concern.

- **Stats `--quiet` returns nil without opening store** (`/Users/leeovery/Code/tick/internal/cli/stats.go:12-14`). While this is a valid optimization (stats with --quiet has no meaningful output), it differs from other commands where --quiet still performs the operation and outputs minimal info (e.g., list --quiet prints IDs). For stats there is no "minimal" output, so this is defensible, but a user might expect --quiet to validate that the tick project exists. Not a bug, but a behavioral inconsistency.

- **`Store.Query` reads and parses JSONL even though queries only use SQLite** (`/Users/leeovery/Code/tick/internal/storage/store.go:230-243`). `readAndEnsureFresh` at line 237 reads the full file, parses all tasks into `[]task.Task`, computes the SHA256 hash, then discards the parsed tasks -- only the cache freshness matters for reads. The parsed `[]task.Task` return value is discarded (assigned to `_`). This is correct for freshness, but the full parse is wasted work on reads. The hash computation alone would suffice for read-path freshness checks.

GAPS:

- **Usage help is minimal** (`/Users/leeovery/Code/tick/internal/cli/app.go:203-208`). `printUsage()` only lists the "init" command. All other commands (create, list, show, update, start, done, cancel, reopen, ready, blocked, dep, stats, rebuild) are undiscoverable from `tick` or `tick help`. No `--help`/`-h` flag is recognized either. While the primary audience is AI agents (which read specifications), human users running `tick` for the first time would see only "init" and assume the tool has minimal functionality.

- **No `--help` per-command**. There is no mechanism to print help for individual commands like `tick create --help`. Unknown flags are silently skipped. This means typos in flag names (e.g., `--priorty`) are silently ignored rather than producing errors, which could lead to confusing behavior where a user thinks they set a value but it was not applied.

- **`StubFormatter` exists in production code** (`/Users/leeovery/Code/tick/internal/cli/format.go:148-171`). This was a scaffolding artifact from Phase 1. It is never used in production paths (NewFormatter always returns a concrete formatter), but its presence in the production package could confuse readers. It could be moved to a test file.

- **No `--blocked-by` flag on `tick update`**. The `update` command supports `--blocks` (make other tasks blocked by this one) but not `--blocked-by` (add blockers to this task). The `create` command supports both `--blocked-by` and `--blocks`. To add a blocker to an existing task, the user must use `tick dep add`. This asymmetry is functional but may surprise users who expect update to have the same flags as create. The plan's Phase 6 acknowledges this partially (tick-core-6-1: "Add dependency validation to create and update --blocked-by/--blocks").

STRENGTHENING:

- **Read-path performance**: The `Query` path parses all JSONL into `[]task.Task` only to discard the result. A dedicated "hash-only" freshness check that skips parsing would roughly halve the CPU cost of every read operation. For small task lists this is negligible; for larger ones it matters.

- **WAL mode for SQLite**: The cache opens SQLite in default journal mode. Enabling WAL (`PRAGMA journal_mode=WAL`) in `OpenCache` would improve concurrent read performance, which is relevant since the store uses shared read locks.

- **Error messages for structured output**: All errors go to stderr as plain text (`fmt.Fprintf(a.Stderr, "Error: %s\n", err)` at `/Users/leeovery/Code/tick/internal/cli/app.go:94`). When `--json` is active, callers might expect errors in JSON format on stderr for programmatic parsing. This is spec-compliant as-is but limits machine consumption of errors.

- **Dependency validation on update `--blocks`**: When `update --blocks` is used, `applyBlocks` modifies target tasks' BlockedBy arrays, then `ValidateDependency` checks each new dependency. But the validation only checks cycle detection for `(blockID, opts.id)` -- the "block target blocked by this task" direction. It does not check whether the update command's own task (which may have had its parent changed in the same update) creates a child-blocked-by-parent violation. This edge case (updating parent and blocks simultaneously in a way that creates a child-blocked-by-parent) is unlikely but theoretically possible.

NEXT_STEPS:

- **Complete Phases 6-8**: The plan documents 14 pending tasks across three analysis-fix phases. These address known technical debt: dependency validation gaps, code duplication, dead code removal, and spec compliance. These should be the immediate priority.

- **Add comprehensive help text**: Expand `printUsage()` to list all commands. Add `--help` / `-h` support for global and per-command usage. This is critical for human usability.

- **Route `log.Printf` through injected writers**: Replace the three `log.Printf` calls in `store.go` with a warning callback that routes through the app's stderr writer. This enables testability and consistent output formatting.

- **Fix `truncateTitle` for Unicode**: Change to rune-aware truncation to avoid producing invalid UTF-8. Small fix with clear correctness impact.

- **Consider `tick delete` / `tick archive`**: The current system has no way to remove tasks. Over time, completed/cancelled tasks accumulate in JSONL. While this is acceptable for the current scope, a cleanup mechanism will eventually be needed.

SUMMARY:

Tick's core implementation (Phases 1-5) delivers a well-architected, thoroughly tested task tracker. The dual-storage design (JSONL source of truth + SQLite query cache with SHA256 freshness) is sound and correctly implemented. The atomic write pipeline (temp + fsync + rename), file locking (shared reads / exclusive writes), and cache corruption recovery are all robust. Test coverage is comprehensive with table-driven tests, integration tests, race detector verification, and cross-format consistency checks. The three output formats (TOON, Pretty, JSON) are cleanly separated behind a Formatter interface with shared base behavior.

The product is functional and correct for its primary audience (AI coding agents). The main areas for improvement are: (1) the three `log.Printf` calls that bypass the injected writer architecture, (2) the byte-vs-rune `truncateTitle` bug, (3) minimal help/usage text, and (4) the 14 pending analysis-fix tasks in Phases 6-8 that address known code duplication and validation gaps. None of these are blocking for initial use, but Phases 6-8 should be completed before considering the product fully hardened.
