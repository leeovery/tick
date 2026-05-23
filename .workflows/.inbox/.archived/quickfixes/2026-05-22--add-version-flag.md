# Add --version flag

Tick currently exposes its version through a `version` subcommand. Add a `--version` flag as an alternative entry point so users can run `tick --version` and get the same output. This is a common convention for CLIs and people reach for it instinctively.

Both forms should work and produce identical output — the flag is purely an alternative invocation, not a replacement. The existing `version` subcommand stays as-is.

The change touches flag parsing / dispatch in `internal/cli` (likely `app.go` or the top-level argument handling before subcommand dispatch). The version string itself is already wired up via the `Version` variable injected at build time through ldflags, so no changes are needed to how the value is sourced.
