# Analysis — Architecture (Cycle 1)

AGENT: architecture
STATUS: findings
FINDINGS_COUNT: 1

FINDINGS:

- FINDING: globalFlagSet omits `--version`/`-V`, creating a latent registry inconsistency
  - SEVERITY: low
  - FILES: internal/cli/flags.go:20-30, internal/cli/app.go:418
  - DESCRIPTION: `applyGlobalFlag` now recognises `--version`/`-V` as global flags, but `globalFlagSet` in flags.go (the registry `ValidateFlags` consults to treat an arg as a valid global flag for any command) lists every other global flag — including `--help`/`-h` — yet omits both `--version` and the new `-V`. This is not functionally triggered today: `parseArgs` strips global flags into the `flags` struct before dispatch, and `flags.version` short-circuits at app.go:45 before `ValidateFlags` runs, so a stray `--version`/`-V` never reaches the registry. But the two parallel sources of truth (the `applyGlobalFlag` switch vs. the `globalFlagSet` map) now disagree about what counts as a global flag — exactly the kind of drift this codebase guards against elsewhere (e.g. the `commandFlags` drift-detection test). If validation ordering ever changes, or `globalFlagSet` gains a second consumer, the omission becomes a real "unknown flag" bug for `-V`. The work unit touched the version-flag surface, so aligning the registry here is in-scope and cheap.
  - RECOMMENDATION: Add `"--version": true` and `"-V": true` to `globalFlagSet` so both definitions of the global-flag set stay consistent (this also subsumes the pre-existing `--version` gap).

SUMMARY: The implementation is a clean, correct, mechanical change with good seam quality — `-V` reuses the existing `flags.version` short-circuit path, there is no `-v`/`-V` collision, and tests assert exact byte-identical output vs. the `version` subcommand. The only architectural note is the registry inconsistency above, currently masked by dispatch ordering.
