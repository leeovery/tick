# Homebrew Tap for Tick

Install tick via Homebrew on macOS.

## Usage

```sh
brew tap leeovery/tick
brew install tick
```

## Releasing a New Version

To publish a new version of tick to the tap:

1. Set `version` in `Formula/tick.rb` to the new version number (without `v` prefix, e.g., `1.2.3`)
2. Update the `sha256` for each architecture from the release assets
