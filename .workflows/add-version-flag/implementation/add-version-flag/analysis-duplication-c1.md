AGENT: duplication
STATUS: findings
FINDINGS_COUNT: 1

FINDINGS:
- FINDING: Version output string literal duplicated across two dispatch branches
  SEVERITY: low
  FILES: internal/cli/app.go:39, internal/cli/app.go:50
  DESCRIPTION: The exact formatted output `fmt.Fprintf(a.Stdout, "tick version %s\n", Version)` is written inline in both the `--version` global-flag short-circuit (line 39) and the `version` subcommand branch (line 50). Byte-for-byte parity is enforced by a test (`cli_test.go:517` "it produces identical output to the version subcommand"), so any future change to the format string must be made in both places or parity silently breaks. Single-sourcing the string would make the parity contract structural rather than test-enforced.
  RECOMMENDATION: Extract a tiny unexported helper such as `func printVersion(w io.Writer) { fmt.Fprintf(w, "tick version %s\n", Version) }` in app.go and call it from both the `flags.version` branch and the `version` subcommand branch.

SUMMARY: One low-severity duplication: the `"tick version %s\n"` print is inlined in both the `--version` global flag handler and the `version` subcommand handler in app.go; extracting a one-line helper would single-source the format string the parity test depends on.
