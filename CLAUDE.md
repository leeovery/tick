# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Tick is a CLI task management tool written in Go. It stores tasks in JSONL (append-only source of truth) with a SQLite cache for fast queries. File locking (gofrs/flock) ensures safe concurrent access.

## Commands

```bash
# Build
go build -o tick ./cmd/tick/

# Test
go test ./...                          # all tests
go test ./internal/cli                 # single package
go test ./internal/cli -run TestInit   # single test
go test ./internal/storage -count=1    # no cache

# Lint
go vet ./...
gofmt -w ./internal ./cmd
golangci-lint run ./...   # config in .golangci.yml (standard set + modernize)
```

## Architecture

```
cmd/tick/main.go          → entry point, injects Stdout/Stderr/Getwd/IsTTY into cli.App
internal/cli/             → command handlers, flag parsing, formatters (Toon/Pretty/JSON)
internal/task/            → domain model (Task, Status, StateMachine, cascades, transition history)
internal/storage/         → JSONL persistence + SQLite cache, file locking via .tick/lock
internal/doctor/          → diagnostic checks (JSONL syntax, dependency cycles, cache staleness)
internal/migrate/         → import framework: Provider interface + Engine for external tool migration
internal/testutil/        → shared test helpers (FindRepoRoot)
scripts/install.sh        → platform-aware installer (Homebrew on macOS, binary download on Linux)
release                   → release script with AI-generated notes via Claude CLI
```

**Data flow:** `App.Run(args)` → parse flags → resolve format → dispatch to `Run<Command>(dir, fc, fmtr, flagArgs, literals, stdout)` → `Store.Mutate/Query` → JSONL + SQLite

**Storage model:** `.tick/` directory in project root contains `tasks.jsonl` (source of truth), `cache.db` (SQLite, rebuilt from JSONL via SHA256 hash comparison + schema version check), and `lock` (flock). Schema version stored in metadata table; mismatch triggers delete+rebuild.

## Key Patterns

- **DI via struct fields:** App injects Stdout, Stderr, Getwd, IsTTY. Store uses functional options (`StoreOption`).
- **Handler signature:** `Run<Command>(dir string, fc FormatConfig, fmtr Formatter, flagArgs, literals []string, stdout io.Writer) error` — `App.Run` splits arguments at the end-of-flags marker once and hands both halves to every handler that takes arguments (`init`, `stats`, `rebuild` take none); fully-positional commands concatenate them in order.
- **Formatter interface:** `Formatter` with methods FormatTaskList, FormatTaskDetail, FormatCascadeTransition, FormatDepChange, FormatDepTree, FormatStats, FormatMessage, FormatRemoval. Three implementations: ToonFormatter, PrettyFormatter, JSONFormatter. Document-producing methods return `(string, error)`; handlers write through `printDocument()` in `helpers.go`, which prints nothing on error. TOON output is library-encoded end to end (no hand-built sections) — a value the encoder refuses (C0 control char other than tab/newline/CR) fails the command with a diagnostic naming the field or section and the task, rather than emitting a document without it.
- **Status change output:** every command that moves statuses renders one `changed{id,title,from,to,auto}` table in toon/JSON (`StatusChanges.Rows()` merges cascade blocks so a task appears at most once). `start`/`done`/`cancel`/`reopen` return only the table; `create`/`update` pass a non-nil `*StatusChanges` to `outputMutationResult` so the detail document always carries a `changed` section (count-zero when nothing moved); `note add`/`remove` pass nil and carry none. Pretty keeps its transition line and cascade tree.
- **Field selection:** `show --field/--fields` is resolved by the `showFields` registry in `show_fields.go` (names are the detail document's own keys; list sections accept a `.N` position). One resolved value prints bare and ignores the format flags; anything else is the detail document narrowed via `TaskDetail.Fields`, in normal section order. `--quiet` with a selection is refused.
- **Format auto-detection:** TTY → pretty, non-TTY → toon. Override with `--toon`, `--pretty`, `--json`.
- **Error wrapping:** `fmt.Errorf("context: %w", err)` throughout.
- **Task IDs:** `tick-` prefix + 6 hex chars (3 random bytes). Partial ID matching supported (unique prefix resolves to full ID).
- **Task fields:** Title, Status, Priority, Description, Parent, Dependencies (blocked-by/blocks), Type (bug/feature/task/chore), Tags (kebab-case labels), Refs (external links), Notes (timestamped annotations), Transitions (history of status changes).
- **Status transitions:** open → in_progress → done/cancelled, cancel from any status, reopen (done/cancelled → open). Managed by `StateMachine` struct in `state_machine.go`.
- **Cascade rules:** Status changes auto-propagate through parent/child hierarchies. Two public entry points: `ApplyUserTransition` (auto=false on primary target) and `ApplySystemTransition` (auto=true on primary target); both delegate to unexported `applyWithCascades`. Rule 2: start cascades up (open ancestors → in_progress). Rule 3: completion cascades up (all children terminal → parent auto-completes). Rule 4: done/cancel cascades down (non-terminal descendants follow). Rule 5: reopen cascades up (done ancestors → open). Rule 6: adding child to done parent reopens it. Rule 7: cannot add child to cancelled parent. Rule 8: cannot depend on cancelled task. Rule 9: cannot reopen under cancelled parent.
- **Transition history:** `TransitionRecord` tracks from/to/at/auto on each task. Stored in JSONL and `task_transitions` SQLite table. Auto flag distinguishes user-initiated (auto=false via `ApplyUserTransition`) from system-initiated transitions (auto=true via `ApplySystemTransition`); cascade transitions are always auto=true.
- **Ready/blocked queries:** `query_helpers.go` defines `ReadyNo*()` SQL helpers composed into `ReadyConditions()` and `BlockedConditions()` (De Morgan inverse). Ancestor blocking uses a recursive CTE walking the parent chain.
- **Tag filtering:** AND (comma-separated in one `--tag`) / OR (multiple `--tag` flags) composition via SQL subqueries.
- **Cache schema versioning:** `schemaVersion` constant in `cache.go` (currently v3); `ensureFresh()` checks version before freshness hash — mismatch triggers delete+recreate+rebuild.
- **Flag validation:** `ValidateFlags()` in `flags.go` rejects unknown flags before store access. Central `commandFlags` registry maps each command to its valid flags. `ready`/`blocked` flag sets derived from `list` via `copyFlagsExcept()` to prevent drift. Drift-detection test ensures `commandFlags` stays in sync with the help registry.
- **Flag value spellings:** a value-taking flag accepts `--flag value` and `--flag=value`. `cutFlagValue()` in `flags.go` cuts a flag-shaped argument at its first `=`; `flagScanner` walks a parser's flag arguments and resolves the value from either spelling. Each parser that holds value cases switches on the cut name and records the whole argument for positionals, so text carrying `=` or spelling a flag survives intact. Global flags stay exact-match in `applyGlobalFlag`.
- **Tests:** stdlib `testing` only (no testify), `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers. Toon/JSON assertions decode the output and check values (`decodeToonDoc`, `toonRows`, `assertToonFields` in `test_helpers_test.go`); pretty keeps golden-string assertions.
- **Conformance inventory:** `conformance_test.go` lists every document the CLI produces (`conformanceDocs`) and decodes each with a real TOON reader and a strict single-value JSON decoder. `TestConformanceInventoryCoversEveryCommand` requires every key of `commandFlags` to be claimed exactly once by the must-parse, prose or out-of-scope set under both drivers — a new command needs an inventory entry or a declared exemption.
- **README samples are tested:** `readme_samples_test.go` renders each `$ tick …` fence in README.md against a seeded project and compares byte-for-byte (`readmeSampleGroups`). Every prompted fence must be claimed by a `readmeSample` or listed in `readmeSampleExemptions`; the `show` section must document every flag `tick help show` lists; the Global Flags block must list `--`. Changing formatter output or a README sample means updating both.

## Task Management (Dogfooding)

When using Tick for task management in this project (e.g., workflow planning with the `tick` output format), always use the Homebrew-installed `tick` CLI — never `go build` or `./tick`. We're dogfooding our own tool, but the local source may be mid-edit and unbuildable.

## Release & Distribution

- Version injected at build time via ldflags (`-X github.com/leeovery/tick/internal/cli.Version={{.Version}}`); defaults to `"dev"`
- goreleaser builds static binaries (CGO_ENABLED=0) for darwin/linux × amd64/arm64
- Archives named `tick_{version}_{os}_{arch}.tar.gz`
- macOS install: `brew install leeovery/tools/tick` (formula lives in separate `homebrew-tools` repo, updated via GitHub Actions `repository_dispatch`)
- Linux install: `scripts/install.sh` downloads from GitHub releases
