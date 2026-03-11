AGENT: standards
FINDINGS:
- FINDING: ready/blocked exclude both --ready and --blocked, spec only excludes one each
  SEVERITY: low
  FILES: internal/cli/flags.go:93, internal/cli/flags.go:94
  DESCRIPTION: The spec's flag inventory says ready accepts "same as list minus --ready" and blocked accepts "same as list minus --blocked". The implementation excludes both --ready AND --blocked from both commands (copyFlagsExcept with two exclusions). This means `tick ready --blocked` is rejected as unknown, even though the spec implies it should be accepted. The flag count tests confirm 6 flags for each (list has 8, minus 2), not 7 (list minus 1). This is arguably the right behavior since --blocked on the ready command is logically contradictory, but it diverges from the literal spec text.
  RECOMMENDATION: Accept as-is. The stricter behavior prevents user confusion. If strict spec compliance is desired, change each copyFlagsExcept call to exclude only the self-referencing flag.

SUMMARY: Implementation conforms to specification with one low-severity divergence: ready/blocked commands are stricter than spec by excluding both filter flags instead of only their own. All other spec requirements -- central validation, error format, two-level command qualification, excluded commands, dep rm rename, migrate equals-sign removal, pre-subcommand rejection, cleanup of silent-skip logic -- are correctly implemented.
