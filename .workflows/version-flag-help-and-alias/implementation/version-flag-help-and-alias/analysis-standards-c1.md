# Analysis — Standards (Cycle 1)

AGENT: standards
STATUS: clean (low-severity cosmetic note only; spec conformance clean)
FINDINGS_COUNT: 1

FINDINGS:

- FINDING: -V and --version absent from globalFlagSet (consistency, not a regression)
  - SEVERITY: low
  - FILES: internal/cli/flags.go:20-30
  - DESCRIPTION: The new `-V` alias (and the pre-existing `--version`) are not listed in `globalFlagSet`, the defensive map `ValidateFlags` consults to accept global flags appearing in `subArgs`. In practice this is harmless: `parseArgs` runs `applyGlobalFlag` over all args and strips `--version`/`-V` before dispatch, so they never reach `ValidateFlags`. The spec scoped only `help.go` and `app.go` and explicitly excluded any dispatch/precedence change, so leaving `flags.go` untouched conforms to scope. The omission is pre-existing for `--version`, and the implementation simply mirrors that treatment for `-V`, which is internally consistent. Noted only because the map's doc comment claims it contains "all global flags," now slightly more inaccurate.
  - RECOMMENDATION: Optional and out of strict spec scope: add `"--version": true` and `"-V": true` to `globalFlagSet` to match the comment and harden against any future code path that validates before stripping. Not required for correctness; skip if avoiding scope creep on a quick-fix.

SUMMARY: The implementation conforms to every decision point in the spec — `printTopLevelHelp` lists `--help, -h` first and `--version, -V` last using the existing `--quiet, -q` inline-alias style; `printAllHelp` appends `--version/-V` to the global-flags line; `applyGlobalFlag` extends `case "--version":` to `case "--version", "-V":` with no dispatch change; and both required tests exist. `go test ./internal/cli/`, `go vet`, and `gofmt -l` are all clean. No MUST DO / MUST NOT DO violations.
