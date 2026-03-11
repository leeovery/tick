AGENT: architecture
FINDINGS:
- FINDING: Duplicate flag knowledge between commandFlags and help commands registries
  SEVERITY: medium
  FILES: internal/cli/flags.go:33, internal/cli/help.go:26
  DESCRIPTION: Two independent registries define flag information for the same commands -- `commandFlags` in flags.go maps commands to their valid flags with type metadata, and `commands` in help.go lists flags with display metadata (name, arg hint, description). These must be kept in sync manually. Adding a flag to one without the other means either help text is wrong or validation rejects a valid flag. There is no compile-time or test-time check ensuring the two registries agree on which flags exist for each command.
  RECOMMENDATION: Add a test that iterates `commandFlags` entries and verifies each flag name appears in the corresponding `commandInfo.Flags` slice (and vice versa). This is a lightweight guard that catches drift without restructuring the registries. A more structural fix would be to derive one from the other, but the test is proportional to the current codebase size.

- FINDING: ready/blocked flag sets duplicated from list instead of derived
  SEVERITY: medium
  FILES: internal/cli/flags.go:74-91
  DESCRIPTION: The specification says `ready` accepts "same as list minus --ready" and `blocked` accepts "same as list minus --blocked". The implementation copies the list flag definitions into the ready and blocked entries, minus one flag each. If a filter flag is added to `list` in the future, `ready` and `blocked` must be updated independently. This violates the "Compose, Don't Duplicate" principle from code-quality.md -- the blocked set should be derived from list by removing --blocked, not authored independently.
  RECOMMENDATION: Compute `ready` and `blocked` flag sets programmatically from the `list` entry in an init() function. For example: copy list's map, delete "--ready" for the ready command, delete "--blocked" for the blocked command. This ensures the mathematical relationship (ready = list - {--ready}) is self-maintaining.

- FINDING: globalFlagSet duplicates applyGlobalFlag knowledge
  SEVERITY: low
  FILES: internal/cli/flags.go:20-30, internal/cli/app.go:384-402
  DESCRIPTION: The set of global flags is defined in two places: `globalFlagSet` (a map used by ValidateFlags to skip globals) and `applyGlobalFlag` (a switch statement used by parseArgs to extract globals). Adding a new global flag requires updating both. They serve different purposes but encode the same set of flag names.
  RECOMMENDATION: Derive `globalFlagSet` from the authoritative source, or have `applyGlobalFlag` consult `globalFlagSet` instead of a separate switch. The simplest approach: make `globalFlagSet` the single source of truth and have `applyGlobalFlag` look up the flag name in it, then use a secondary map or switch just for which struct field to set. Alternatively, add a test asserting the two are in sync.

SUMMARY: The implementation architecture is sound overall -- centralized validation with command-exported flags is the right pattern, the seam between parseArgs/qualifyCommand/ValidateFlags is clean, and the two dispatch paths (doctor/migrate early, everything else post-format) are both covered. The main structural concern is triple-maintained flag knowledge (commandFlags, help registry, and globalFlagSet/applyGlobalFlag) with no automated sync checks, which creates a drift risk proportional to how often flags are added.
