# Homebrew Tap for Tick

This directory contains the Homebrew formula for tick. It is the development source of truth for the formula, but `brew tap leeovery/tick` does **not** read from here.

## Deployment Prerequisite: Separate Repository

Homebrew tap discovery expects a repository at `github.com/leeovery/homebrew-tick`. The command `brew tap leeovery/tick` clones that repository -- it does not look inside this monorepo's `homebrew-tap/` directory.

The contents of this directory (specifically `Formula/tick.rb`) must be copied or synced to the separate `homebrew-tick` repository before users can install via Homebrew. Without this step, both `brew tap leeovery/tick` and the macOS path in `scripts/install.sh` (line 66) will fail.

### Who is responsible for the sync?

The release workflow (CI) or a manual step must publish the formula to `github.com/leeovery/homebrew-tick` as part of each release. The exact mechanism (GitHub Actions copy, goreleaser homebrew publisher, manual push) is defined by the release pipeline -- this directory only tracks the source formula.

## Usage (end users)

```sh
brew tap leeovery/tick
brew install tick
```

## Releasing a New Version

1. Update `version` in `Formula/tick.rb` (without `v` prefix, e.g., `1.2.3`)
2. Update the `sha256` for each architecture from the release assets
3. Sync the updated formula to the `github.com/leeovery/homebrew-tick` repository
