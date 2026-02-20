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
```

## Architecture

```
cmd/tick/main.go          → entry point, injects Stdout/Stderr/Getwd/IsTTY into cli.App
internal/cli/             → command handlers, flag parsing, formatters (Toon/Pretty/JSON)
internal/task/            → domain model (Task struct, Status enum, ID generation)
internal/storage/         → JSONL persistence + SQLite cache, file locking via .tick/lock
internal/doctor/          → diagnostic checks (JSONL syntax, dependency cycles, cache staleness)
internal/migrate/         → import framework: Provider interface + Engine for external tool migration
internal/testutil/        → shared test helpers (FindRepoRoot)
scripts/install.sh        → platform-aware installer (Homebrew on macOS, binary download on Linux)
release                   → release script with AI-generated notes via Claude CLI
```

**Data flow:** `App.Run(args)` → parse flags → resolve format → dispatch to `Run<Command>(dir, fc, fmtr, args, stdout)` → `Store.Mutate/Query` → JSONL + SQLite

**Storage model:** `.tick/` directory in project root contains `tasks.jsonl` (source of truth), `cache.db` (SQLite, rebuilt from JSONL via MD5 hash comparison), and `lock` (flock).

## Key Patterns

- **DI via struct fields:** App injects Stdout, Stderr, Getwd, IsTTY. Store uses functional options (`StoreOption`).
- **Handler signature:** `Run<Command>(dir string, fc FormatConfig, fmtr Formatter, args []string, stdout io.Writer) error`
- **Formatter interface:** `Formatter` with methods FormatTaskList, FormatTaskDetail, FormatTransition, FormatDepChange, FormatStats, FormatMessage. Three implementations: ToonFormatter, PrettyFormatter, JSONFormatter.
- **Format auto-detection:** TTY → pretty, non-TTY → toon. Override with `--toon`, `--pretty`, `--json`.
- **Error wrapping:** `fmt.Errorf("context: %w", err)` throughout.
- **Task IDs:** `tick-` prefix + 6 hex chars (3 random bytes).
- **Status transitions:** open → in_progress → done/cancelled, reopen back to open.
- **Ready/blocked queries:** `query_helpers.go` defines `ReadyNo*()` SQL helpers composed into `ReadyConditions()` and `BlockedConditions()` (De Morgan inverse). Ancestor blocking uses a recursive CTE walking the parent chain.
- **Tests:** stdlib `testing` only (no testify), `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers.

## Release & Distribution

- goreleaser builds static binaries (CGO_ENABLED=0) for darwin/linux × amd64/arm64
- Archives named `tick_{version}_{os}_{arch}.tar.gz`
- macOS install: `brew install leeovery/tools/tick` (formula lives in separate `homebrew-tools` repo, updated via GitHub Actions `repository_dispatch`)
- Linux install: `scripts/install.sh` downloads from GitHub releases
