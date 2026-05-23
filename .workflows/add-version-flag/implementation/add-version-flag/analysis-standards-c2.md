AGENT: standards
STATUS: clean
FINDINGS_COUNT: 0
FINDINGS: none

SUMMARY: Implementation conforms to the specification and project conventions. `version` field added to `globalFlags`, `--version` recognised in `applyGlobalFlag`, early dispatch in `App.Run` short-circuits before subcommand handling, existing `Version` variable reused unchanged, and `printVersion` helper guarantees byte-for-byte parity with the `version` subcommand. Test coverage in `TestVersionFlag` exercises the flag, asserts identical output to the subcommand, confirms short-circuit behaviour when combined with other flags, and matches project test conventions.
