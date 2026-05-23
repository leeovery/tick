AGENT: duplication
STATUS: clean
FINDINGS_COUNT: 0
FINDINGS: none

SUMMARY: The c1 duplication finding (inline `"tick version %s\n"` format string in two dispatch branches) is resolved. `printVersion(io.Writer)` at internal/cli/app.go:18-20 is now the single source of truth, called from both the `--version` short-circuit (app.go:46) and the `version` subcommand branch (app.go:57). No significant duplication detected.
