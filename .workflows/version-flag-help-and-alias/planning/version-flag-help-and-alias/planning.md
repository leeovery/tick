# Plan: Version Flag Help And Alias

## Phase 1: Apply Change

Surface `--version` (and `--help`) in `tick --help`, and add a `-V` short alias for `--version`.

#### Tasks
status: approved

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| version-flag-help-and-alias-1-1 | List --version and --help in top-level help | Keep `printAllHelp` inline list in sync (add `--version/-V`) |
| version-flag-help-and-alias-1-2 | Add -V short alias for --version | None — reuses existing `flags.version` dispatch |
