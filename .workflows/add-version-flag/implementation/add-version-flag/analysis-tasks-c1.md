---
topic: add-version-flag
cycle: 1
total_proposed: 1
---
# Analysis Tasks: add-version-flag (Cycle 1)

## Task 1: Extract printVersion helper to single-source version output
status: approved
severity: low
sources: duplication, architecture

**Problem**: The exact line `fmt.Fprintf(a.Stdout, "tick version %s\n", Version)` is inlined in two dispatch branches of `internal/cli/app.go` — the `--version` global-flag short-circuit (line 39) and the `version` subcommand branch (line 50). `TestVersionFlag` in `internal/cli/cli_test.go:517` asserts byte-for-byte equality between the two outputs via `bytes.Equal`, so parity is currently enforced only by a test rather than structurally. Any future change to the format string must be made in both places or parity silently breaks.

**Solution**: Extract a tiny unexported package-level helper that owns the version-print format string, and call it from both branches. Makes byte-for-byte parity structural rather than coincidental.

**Outcome**: A single source of truth for the `"tick version %s\n"` format string. Both the `--version` flag handler and the `version` subcommand call the same helper. Existing `TestVersionFlag` parity test continues to pass; all other tests remain green.

**Do**:
1. In `internal/cli/app.go`, add an unexported helper:
   ```go
   func printVersion(w io.Writer) {
       fmt.Fprintf(w, "tick version %s\n", Version)
   }
   ```
2. Replace inline `fmt.Fprintf(a.Stdout, "tick version %s\n", Version)` at app.go:39 (the `--version` short-circuit) with `printVersion(a.Stdout)`.
3. Replace inline `fmt.Fprintf(a.Stdout, "tick version %s\n", Version)` at app.go:50 (the `version` subcommand) with `printVersion(a.Stdout)`.
4. Run `gofmt -w ./internal` and `go vet ./...`.
5. Run `go test ./internal/cli`.

**Acceptance Criteria**:
- Literal `"tick version %s\n"` appears in exactly one place in `internal/cli/app.go`.
- Both `--version` flag branch and `version` subcommand branch invoke `printVersion(a.Stdout)`.
- `TestVersionFlag` parity test at cli_test.go:517 passes unchanged.
- `go test ./...` green; `go vet ./...` and `gofmt` clean.

**Tests**:
- No new tests required — existing `TestVersionFlag` parity assertion already covers the behaviour; single-sourcing makes parity structurally guaranteed.
- Verify via `go test ./internal/cli -run TestVersionFlag` and full `go test ./...`.
