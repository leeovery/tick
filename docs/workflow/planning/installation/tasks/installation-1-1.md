---
id: installation-1-1
phase: 1
status: completed
created: 2026-01-31
---

# Minimal Go Binary

## Goal

The entire release pipeline (goreleaser, GitHub Actions, install script) depends on having a Go project that compiles successfully. Without a buildable binary, nothing downstream can be configured or tested. This task creates the thinnest possible Go main package — just enough to prove the project compiles and produces a `tick` binary.

## Implementation

1. **Initialize the Go module** at the repository root:
   - Create `go.mod` with module path `github.com/leeovery/tick` (or the actual repository path) and a current Go version (e.g., `go 1.23`).

2. **Create the main package** at `cmd/tick/main.go`:
   - Package `main` with a `main()` function.
   - Print a single line to stdout: `tick version dev` (or similar placeholder).
   - Exit with code 0.

3. **Verify the binary compiles**:
   - Run `go build -o tick ./cmd/tick/` from the repository root.
   - Confirm the `tick` binary is produced and is executable.
   - Run `./tick` and confirm it outputs the expected version string and exits 0.

4. **Add the built binary to `.gitignore`**:
   - Add `/tick` to `.gitignore` so the compiled binary is not committed.

## Tests

- `"go build produces a tick binary without errors"` — run `go build -o tick ./cmd/tick/` and assert exit code 0 and the binary file exists
- `"tick binary outputs version string to stdout"` — execute the built binary and assert stdout contains `tick`
- `"tick binary exits with code 0"` — execute the built binary and assert the exit code is 0

## Edge Cases

None identified for this task. It is a minimal scaffold with no branching logic or external dependencies.

## Acceptance Criteria

- [ ] `go.mod` exists at repository root with a valid module path and Go version
- [ ] `cmd/tick/main.go` exists with a `main()` function in package `main`
- [ ] `go build -o tick ./cmd/tick/` succeeds with exit code 0
- [ ] Executing the built binary prints output to stdout and exits with code 0
- [ ] Built binary (`tick`) is listed in `.gitignore`

## Context

The specification identifies a "tick-core (buildable binary)" as the foundational dependency that blocks all distribution work: goreleaser configuration, GitHub Actions release workflow, and the install script. This task satisfies that dependency at its minimum viable form — a compilable Go binary. The `cmd/tick/` directory convention is standard for Go projects with multiple potential packages and aligns with goreleaser's default expectations for locating the main package.

The specification notes: "goreleaser configuration can be set up early with placeholder build targets" — this task provides that placeholder build target. The binary will be replaced by real application code in a separate topic (tick-core), but the release pipeline only needs something that compiles.

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
