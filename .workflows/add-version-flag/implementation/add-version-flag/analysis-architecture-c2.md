AGENT: architecture
STATUS: clean
FINDINGS_COUNT: 0
FINDINGS: none

SUMMARY: Implementation architecture is sound. `printVersion` helper (app.go:18) centralises the format string so both entry points are byte-identical by construction rather than by test. The `--version` flag follows the established global-flag pattern (struct field in `globalFlags` + case in `applyGlobalFlag`) and dispatches via the same early short-circuit shape as `--help`. Test coverage exercises the cross-task seam (byte-equality with subcommand, short-circuit precedence over `--json`, and no-subcommand path). No structural issues.
