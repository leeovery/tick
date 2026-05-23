# Report: Task 2-1 — Extract printVersion helper to single-source version output

**Task ID**: tick-d15179
**STATUS**: Complete
**FINDINGS_COUNT**: 0

## Acceptance Criteria

- Literal `"tick version %s\n"` appears in exactly one place in `internal/cli/app.go`.
- Both `--version` flag branch and `version` subcommand branch invoke `printVersion(a.Stdout)`.
- `TestVersionFlag` parity test at `cli_test.go:517` passes unchanged.
- `go test ./...` green; `go vet ./...` and `gofmt` clean.

## Spec Context

Spec requires `tick --version` and `tick version` to produce byte-identical output (`tick version {Version}\n`). This task is the cleanup step extracted from the c1 duplication analysis — single-sourcing makes parity structurally guaranteed rather than test-enforced.

## Implementation

- **Status**: Implemented.
- **Location**:
  - Helper: `internal/cli/app.go:15-20` (doc comment + `printVersion(w io.Writer)`).
  - Call site 1 (`--version` short-circuit): `internal/cli/app.go:46` — `printVersion(a.Stdout)`.
  - Call site 2 (`version` subcommand): `internal/cli/app.go:57` — `printVersion(a.Stdout)`.
- **Notes**: Grep across the repo confirms `"tick version %s"` appears in exactly one place in production code (`app.go:19`). Tests at `cli_test.go:548,566,587` reconstruct the expected string from `Version` via concatenation rather than duplicating the format string. Helper carries a clear doc comment explaining intent (structural parity over test-enforced parity).

## Tests

- **Status**: Adequate.
- **Coverage**: `TestVersionFlag` at `internal/cli/cli_test.go:516-552` exercises both code paths and asserts `bytes.Equal` between them, plus a literal expected value check. Sibling subtests at lines 554 and 575 cover `--version` combined with other flags and no-subcommand cases.
- **Notes**: Plan correctly states no new tests required. Not over-tested; parity check is the right granularity.

## Code Quality

- Project conventions: Followed. Unexported package-level helper using `io.Writer`, consistent with project DI pattern of injecting writers into `App`.
- SOLID principles: Good. Single responsibility (format and write the version line).
- Complexity: Low. Two-line helper.
- Modern idioms: Yes. Standard `fmt.Fprintf` against `io.Writer`.
- Readability: Good. Doc comment explains the "why" (structural parity), not just the "what".
- Issues: None.

## Blocking Issues

None.

## Non-Blocking Notes

None.
