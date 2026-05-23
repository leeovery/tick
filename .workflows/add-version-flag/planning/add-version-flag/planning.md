# Plan: Add Version Flag

## Phase 1: Apply Change

Add `--version` as a global flag in `internal/cli/app.go`, dispatching to the same output as the `version` subcommand, and add test coverage.

#### Tasks
status: approved

| Internal ID | Name | Edge Cases |
|-------------|------|------------|
| add-version-flag-1-1 | Wire --version global flag | Flag must short-circuit before subcommand validation; output must match `tick version` byte-for-byte |
