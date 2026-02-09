AGENT: architecture
FINDINGS:
- FINDING: Formatter interface uses interface{} parameters losing type safety at compile time
  SEVERITY: medium
  FILES: /Users/leeovery/Code/tick/internal/cli/format.go:13-25, /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:66-69, /Users/leeovery/Code/tick/internal/cli/pretty_formatter.go:18-21, /Users/leeovery/Code/tick/internal/cli/json_formatter.go:103-106
  DESCRIPTION: Every Formatter method (FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats) accepts `interface{}` as its data parameter and performs a runtime type assertion at the top of each implementation. This means a caller passing the wrong type (e.g. passing []TaskRow to FormatTaskDetail) compiles successfully but fails at runtime. With 3 formatters x 5 methods = 15 assertion sites, the surface area for silent type mismatches is significant. The pattern also prevents Go's type system from catching regressions during refactoring.
  RECOMMENDATION: Replace the single polymorphic Formatter interface with type-specific methods. For example, define `FormatTaskList(w io.Writer, rows []TaskRow) error` and `FormatTaskDetail(w io.Writer, data *showData) error` directly. If a single interface is still desired, use generics or separate interfaces per output kind. This eliminates all 15 runtime assertions and makes misuse a compile error.

- FINDING: Shared data types (showData, relatedTask, listRow) are unexported but consumed across formatters
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/show.go:12-32, /Users/leeovery/Code/tick/internal/cli/list.go:14-19, /Users/leeovery/Code/tick/internal/cli/toon_formatter.go:13-18
  DESCRIPTION: The `showData` and `relatedTask` structs in show.go use unexported fields (e.g. `id`, `title`, `status`). These are passed to formatters via `interface{}`, where each formatter accesses them by direct field reference within the same package. This works because everything is in the `cli` package, but creates an implicit coupling: `showData` is a data contract between command handlers and formatters, yet its fields are unexported and undocumented as a contract. Meanwhile, `TaskRow`, `StatsData`, `TransitionData`, and `DepChangeData` in toon_formatter.go are exported -- creating an inconsistency in the API surface of the same data flow.
  RECOMMENDATION: Either export all formatter data types consistently (export showData and relatedTask alongside the already-exported TaskRow, StatsData, etc.) or keep them all unexported. The mixed approach makes it harder to reason about which types are contracts vs internal details.

- FINDING: Store opens cache.db eagerly in NewStore even for operations that may not need it
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/engine/store.go:56-90
  DESCRIPTION: `NewStore` unconditionally opens the SQLite cache at construction time (line 71-74). For the `init` command, which only creates files and never queries the cache, this means the cache.db file is created as a side effect of Store construction even though `init` does not use the Store at all. This is not a bug since `init` avoids creating a Store, but any new command that needs the tick directory but not the cache would face this. The specification explicitly says "SQLite cache is created on first operation (not at init)." The current code respects this by not using Store in init, but the coupling is fragile.
  RECOMMENDATION: Consider lazy initialization of the cache connection -- open it on first Query/Mutate call rather than in the constructor. This would make the Store safe to construct without side effects on the filesystem.

- FINDING: StubFormatter left in production code
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/format.go:93-127
  DESCRIPTION: The `StubFormatter` type is annotated as a placeholder ("will be replaced by concrete Toon, Pretty, and JSON formatters in tasks 4-2 through 4-4") but remains in the production codebase alongside the actual formatter implementations. It is not referenced anywhere in production code paths (only concrete formatters are instantiated in `newFormatter`), making it dead code.
  RECOMMENDATION: Remove StubFormatter. It served its purpose during incremental development and is now unused.

- FINDING: Repeated store setup pattern across every command handler
  SEVERITY: low
  FILES: /Users/leeovery/Code/tick/internal/cli/create.go:33-44, /Users/leeovery/Code/tick/internal/cli/show.go:43-52, /Users/leeovery/Code/tick/internal/cli/list.go:237-245, /Users/leeovery/Code/tick/internal/cli/transition.go:22-31, /Users/leeovery/Code/tick/internal/cli/dep.go:42-51, /Users/leeovery/Code/tick/internal/cli/stats.go:72-81, /Users/leeovery/Code/tick/internal/cli/update.go:53-64, /Users/leeovery/Code/tick/internal/cli/rebuild.go:12-22
  DESCRIPTION: Every command handler that interacts with storage repeats the same 3-step pattern: DiscoverTickDir, NewStore, defer Close. This is 8 occurrences of identical boilerplate. While not yet crossing the "rule of three" threshold for abstraction (since each is simple), the pattern is so consistent that a helper would reduce noise and ensure consistent error handling.
  RECOMMENDATION: Consider extracting a `withStore(ctx *Context, fn func(*engine.Store) error) error` helper that encapsulates discovery + construction + close. This is a low-priority cleanup that would reduce ~6 lines per command handler.

SUMMARY: The architecture is well-structured overall with clean package boundaries (task model, storage, cache, engine, cli) and a sound layering strategy. The primary structural concern is the Formatter interface using interface{} parameters, which trades compile-time safety for polymorphism across 15 method implementations. The remaining findings are minor: dead stub code, an inconsistent export pattern on data types, and repeated boilerplate that could benefit from a small helper.
