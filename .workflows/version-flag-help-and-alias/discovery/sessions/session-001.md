# Discovery Session 001

Date: 2026-06-04
Work unit: version-flag-help-and-alias

## Description (as of session)

List --version in top-level help and add a -V short alias.

## Seed

- seeds/2026-05-23-list-version-in-help.md (inbox:idea)
- seeds/2026-05-23-version-short-alias.md (inbox:idea)

## Imports

(none)

## Map State at Start

(n/a — single-topic work)

## Exploration

Two ideas promoted from the inbox, both follow-ups to the existing
`--version` flag (from the add-version-flag work unit). The first
asks to surface `--version` in the global-flags section of
`tick --help` (`printTopLevelHelp`), where it is currently omitted
alongside `--help`. The second asks for a `-V` short alias, which
was excluded from the original version-flag spec.

Both were confirmed against the code during shaping: the help text
lives in `printTopLevelHelp` (and `printAllHelp`) in
internal/cli/help.go, and the flag matching lives in
`applyGlobalFlag` in internal/cli/app.go (`case "--version":`).
Both changes are small, mechanical, and co-located in the same two
functions — no behaviour to debate, nothing to diagnose, no design
questions. They were folded into a single quick-fix work unit
rather than two separate ones because of that co-location and
triviality.

## Edits

(none)

## Topics Identified

(none)

## Conclusion

Routed to scoping.
