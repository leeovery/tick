TASK: Minimal Go Binary

ACCEPTANCE CRITERIA:
- [x] `go.mod` exists at repository root with a valid module path and Go version
- [x] `cmd/tick/main.go` exists with a `main()` function in package `main`
- [x] `go build -o tick ./cmd/tick/` succeeds with exit code 0
- [x] Executing the built binary prints output to stdout and exits with code 0
- [x] Built binary (`tick`) is listed in `.gitignore`

STATUS: Complete

SPEC CONTEXT: The specification identifies "tick-core (buildable binary)" as the foundational dependency blocking all distribution work. goreleaser needs a Go project that compiles successfully. This task provides the minimum viable build target so the release pipeline can be configured and tested.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `/Users/leeovery/Code/tick/go.mod` (line 1): module `github.com/leeovery/tick`, Go 1.25.7
  - `/Users/leeovery/Code/tick/cmd/tick/main.go` (lines 1-18): package main with `main()` function
  - `/Users/leeovery/Code/tick/.gitignore` (line 3): `/tick` entry
- Notes: The binary has evolved well beyond the planned "minimal scaffold" -- it now wires up a full `cli.App` with DI of Stdout, Stderr, Getwd, and IsTTY. This is expected since the tick-core topic was implemented after this task. The key point is that the scaffold exists and compiles, which is what this task required.

TESTS:
- Status: Adequate
- Coverage:
  - `/Users/leeovery/Code/tick/cmd/tick/build_test.go` (lines 13-55): `TestBuild` with three subtests
  - "go build produces a tick binary without errors" -- builds binary, checks file exists, non-empty, executable permissions
  - "tick binary outputs version string to stdout" -- runs binary, checks stdout contains "tick"
  - "tick binary exits with code 0" -- runs binary, checks no error (exit 0)
- Notes: All three planned test cases from the task are implemented. Tests are focused and not redundant. The build is done once at the top and shared across subtests, which is efficient. The test for "version string" checks for "tick" in stdout -- when run without arguments, `printUsage()` outputs "Usage: tick <command> [options]" which satisfies this. Uses `testutil.FindRepoRoot` helper with `t.Helper()` per project conventions.

CODE QUALITY:
- Project conventions: Followed -- stdlib `testing` only, `t.Run()` subtests, `t.TempDir()` for isolation, `t.Helper()` on helpers, DI via struct fields, package doc comment on testutil
- SOLID principles: Good -- main.go is thin, delegating to cli.App; single responsibility maintained
- Complexity: Low -- main.go is 18 lines, build_test.go is 55 lines, both straightforward
- Modern idioms: Yes -- uses `exec.Command`, `t.TempDir()` for auto-cleanup, `CombinedOutput` for build diagnostics
- Readability: Good -- clear test names matching the planned test descriptions, intent is obvious
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The task plan mentions printing "tick version dev" as the placeholder output, but the actual binary prints usage text. This is fine since the acceptance criteria only require "prints output to stdout and exits with code 0", and the binary has evolved past the placeholder stage through the tick-core topic.
