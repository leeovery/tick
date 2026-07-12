# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] - 2026-03-22

🗑️ Removed — Deleted the internal `IDEAS.md` and `bugs.md` planning notes from the repo.

🐛 Fixed — Status-change output no longer lists sibling/descendant tasks that were already done or cancelled and didn't change.

## [0.2.8] - 2026-07-12

🔧 Changed
- Note text limit raised from 500 to 2000 characters.

## [0.2.7] - 2026-06-04

✨ Added
- Add `-V` as a short alias for `--version`.

🔧 Changed
- `tick --help` now also lists `--help` in the global flags list.

## [0.2.6] - 2026-06-03

✨ Added
- `tick ready` now includes unblocked in-progress tasks, not just open ones — resuming interrupted work is treated as "ready" too.
- Ready results float in-progress tasks to the top ahead of priority ordering, so resuming existing work is prioritized over starting new work.

🔧 Changed
- `tick blocked` now reports blocked in-progress tasks in addition to blocked open ones.
- Stats' ready/blocked counts now account for in-progress tasks, fixing an edge case where blocked could have gone negative.

## [0.2.5] - 2026-05-23

✨ Added
- Add a `--version` global flag that prints the version and exits, equivalent to the `version` subcommand — works even before or without a subcommand.

## [0.2.4] - 2026-04-30

Maintenance release — no notable source changes
## [0.2.3] - 2026-03-30

🔧 Changed
- Modernized the codebase to Go's `any` type throughout, replacing `interface{}` — no user-facing effect.
- `note tree` is no longer misinterpreted as a dep-tree command — it now correctly reports an unknown sub-command error instead of silently doing the wrong thing.

## [0.2.2] - 2026-03-28

✨ Added
- New `tick dep tree` command visualizes dependency chains — run with no argument for the full graph, or with a task ID for a focused upstream/downstream view.

🔧 Changed
- `tick dep tree` output now renders as a box-drawing tree in Pretty mode and a flat edge list in TOON/JSON mode, with cycle protection against corrupted circular dependency data.

## [0.2.0] - 2026-03-14

✨ Added
- Unknown or misspelled flags on any command are now rejected with a clear error instead of silently ignored.
- Reparenting a task away from a done-eligible parent re-evaluates completion — the old parent auto-completes if its remaining children are all terminal.
- `dep add`/`dep remove` now validate dependencies on cancelled tasks too.

🔧 Changed
- `dep rm` renamed to `dep remove` (matches `note remove`); the old `rm` alias is no longer recognized.
- `migrate --from=value` (equals-sign syntax) is no longer accepted — use `migrate --from value` instead.

🐛 Fixed
- Adding a child task to a done parent now records the parent's auto-reopen as system-initiated (`auto=true`) instead of incorrectly marking it as a manual transition.

## [0.1.3] - 2026-03-07

✨ Added
- Documented transition and cascade output examples for `start`, `done`, `cancel`, and `reopen` — shows how TOON, Pretty, and JSON formats confirm status changes and cascaded updates to related tasks.

## [0.1.2] - 2026-03-07

✨ Added
- Status changes now cascade through parent/child hierarchies — starting a subtask auto-starts its ancestors, completing or cancelling a parent cascades to open children, and finishing all of a parent's children auto-completes it.
- Reopening a task auto-reopens its done ancestors, and adding a new child to a done parent automatically reopens that parent.
- Task status changes are now recorded as a timestamped transition history, distinguishing user-initiated from auto-cascaded transitions.
- `tick dep add` and task creation now block adding a cancelled task as a dependency or parent, guiding you to reopen it first.

🔧 Changed
- Reparenting a task now auto-completes its old parent when all remaining siblings are terminal, and reopens the new parent if it was done.
- `tick start/done/cancel/reopen` output now shows the full cascade (primary transition plus any auto-triggered child/parent changes) across toon, pretty, and JSON formats.
- Cache schema bumped to v2 to store transition history, triggering an automatic cache rebuild.

## [0.1.1] - 2026-03-01

✨ Added
- New `--type` flag (bug/feature/task/chore) on `create`, `list`, `update`, `ready`, and `blocked` — categorize and filter tasks by type.
- New `--tags` and `--refs` flags on `create` and `update` — attach kebab-case labels and external links (URLs, issue keys) to tasks.
- New `note add`/`note remove` subcommands — attach timestamped notes to a task and remove them by index.
- Tag filtering with AND/OR composition — `--tag a,b` for AND, repeated `--tag` flags for OR.
- New `--count` flag on `list`, `ready`, and `blocked` — cap the number of results returned.
- Partial task ID matching — any unique ID prefix resolves automatically everywhere an ID is accepted.

🔧 Changed
- `show` output now includes type, tags, refs, and notes alongside existing task detail.
- SQLite cache is now versioned and auto-rebuilds when the schema changes or the version is missing, preventing stale or incompatible caches from causing query errors.

## [0.1.0] - 2026-03-01

✨ Added
- Task type field (bug, feature, task, chore) — set with `--type` on create/update, filter with `--type` on list/ready/blocked, shown in list and show output.
- Kebab-case tags — attach with `--tags`/`--clear-tags`, filter with `--tag` (comma-separated for AND, repeated flags for OR), shown in task detail.
- External reference links — attach with `--refs`/`--clear-refs` on create/update, shown in task detail.
- Timestamped notes — add and remove them with `tick note add`/`tick note remove`.
- Partial task IDs — most commands now accept any unique ID prefix (3+ hex chars) instead of requiring the full ID.
- `--count` flag on list/ready/blocked to cap the number of results returned.

## [0.0.9] - 2026-02-26

✨ Added
- Add a `version` command to print the installed tick version.

🔧 Changed
- Release builds now embed the actual version number instead of always reporting a dev build.

## [0.0.8] - 2026-02-20

✨ Added
- `tick remove` command to permanently delete tasks, cascading to descendants and cleaning up dependency references on survivors.
- `--clear-description` flag on `tick update` to remove a task's description.
- `tick help --all` for a full reference of all commands and flags.

🔧 Changed
- `--ready`/`--blocked` (and their `ready`/`blocked` aliases) now also account for a dependency-blocked ancestor when determining readiness.
- SQLite cache file renamed from `.tick/.store` to `.tick/cache.db` — update your `.gitignore` accordingly.

## [0.0.7] - 2026-02-20

✨ Added
- `tick ready`/`tick blocked`/`list --ready`/`list --blocked` now trace the full ancestor chain — a task with a dependency-blocked parent or grandparent no longer shows up as ready, and appears as blocked instead.

## [0.0.6] - 2026-02-19

✨ Added
- `remove` command permanently deletes one or more tasks from a project, with cascade deletion of children, automatic cleanup of dependency references, and an interactive confirmation prompt (bypassable with `--force`/`-f`).
- `tick help remove` documents usage, flags, cascade behavior, and Git-based recovery for accidental removals.

## [0.0.5] - 2026-02-17

✨ Added
- Remove a task's description with `--clear-description` on `tick update`.

🔧 Changed
- Descriptions are now trimmed of leading/trailing whitespace on create and update.
- `tick update --description ""` no longer silently clears the description — it now errors and tells you to use `--clear-description` instead.

## [0.0.4] - 2026-02-15

✨ Added
- `tick doctor` — run 10 diagnostic checks (JSONL syntax, ID format, duplicate IDs, orphaned parents/dependencies, self-references, dependency cycles, child-blocked-by-parent, parent-done-with-open-children, cache staleness) with clear pass/fail markers and fix suggestions, read-only and never fixes anything itself.
- `tick migrate --from beads` — import tasks from beads, with `--dry-run` to preview and `--pending-only` to skip already-completed items; unknown providers list the available ones.
- goreleaser + GitHub Actions release pipeline — pushing a `vX.Y.Z` tag now builds and publishes binaries for macOS and Linux (amd64/arm64).
- `scripts/install.sh` — one-line install via `curl -fsSL ... | bash`, delegating to Homebrew on macOS and downloading the right binary on Linux.
- `tick help` / `tick help <command>` / `tick --help` / `tick -h` — full command reference with per-command flags, plus `tick help --all` for a compact one-shot listing.
- Top-level README with install instructions, command reference, and output format examples.

🔧 Changed
- Swapped the `mattn/go-sqlite3` CGO driver for the pure-Go `modernc.org/sqlite`, simplifying cross-compilation for release builds.

## [0.0.3] - 2026-02-15

Maintenance release — no notable source changes
## [0.0.2] - 2026-02-15

🐛 Fixed
- Release checksum lookup now reads the local build output instead of re-downloading from GitHub, fixing a race where the checksums file wasn't yet available.

## [0.0.1] - 2026-02-15

Initial release.
