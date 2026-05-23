AGENT: architecture
STATUS: findings
FINDINGS_COUNT: 1

FINDINGS:
- FINDING: Version print logic duplicated across two dispatch branches
  SEVERITY: low
  FILES: internal/cli/app.go:38-41, internal/cli/app.go:49-52
  DESCRIPTION: The exact `fmt.Fprintf(a.Stdout, "tick version %s\n", Version)` line now appears in two places — the new `--version` short-circuit and the existing `version` subcommand branch. `TestVersionFlag` asserts byte-for-byte equality via `bytes.Equal(flagStdout.Bytes(), subStdout.Bytes())`, meaning the two callsites must stay synchronised forever. This is the "parallel computations that must be kept in sync" anti-pattern called out in code-quality.md. With only two callsites it sits under the Rule of Three, so it is a low-priority observation rather than a must-fix — but a trivial helper would make the byte-for-byte contract structural rather than coincidental.
  RECOMMENDATION: Extract a tiny package-level helper (e.g. `func printVersion(w io.Writer) { fmt.Fprintf(w, "tick version %s\n", Version) }`) and call it from both branches.

SUMMARY: Implementation composes cleanly into the existing global-flag parser and dispatch flow with no structural concerns. Sole observation is minor duplication of the version-print line across the `--version` and `version` branches.
