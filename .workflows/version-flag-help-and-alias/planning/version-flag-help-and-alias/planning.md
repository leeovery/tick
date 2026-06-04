# Plan: Version Flag Help And Alias

## Phase 1: Apply Change

Surface `--version` (and `--help`) in `tick --help`, and add a `-V` short alias for `--version`.

#### Tasks
status: approved

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| version-flag-help-and-alias-1-1 | List --version and --help in top-level help | Keep `printAllHelp` inline list in sync (add `--version/-V`) |
| version-flag-help-and-alias-1-2 | Add -V short alias for --version | None — reuses existing `flags.version` dispatch |

### Phase 2: Analysis (Cycle 1)

Address findings from Analysis (Cycle 1).

#### Tasks
status: approved

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| version-flag-help-and-alias-2-1 | Add --version and -V to globalFlagSet | Subsumes pre-existing `--version` gap; extend drift-detection test if one covers `globalFlagSet` membership |
| version-flag-help-and-alias-2-2 | Collapse duplicated version-flag tests into a table-driven case | Preserve coverage of both `--version` and `-V`; assert byte-identical output to `version` subcommand |
