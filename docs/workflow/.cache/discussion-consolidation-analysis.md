---
checksum: 43cdc9b4e58540e7971028124df6456e
generated: 2026-01-20T21:05:00Z
discussion_files:
  - archive-strategy-implementation.md
  - cli-command-structure-ux.md
  - config-file-design.md
  - data-schema-design.md
  - doctor-command-validation.md
  - freshness-dual-write.md
  - hierarchy-dependency-model.md
  - id-format-implementation.md
  - installation-options.md
  - migration-subcommand.md
  - toon-output-format.md
  - tui.md
---

# Discussion Consolidation Analysis

## Recommended Groupings

### Core Data & Storage
- **data-schema-design**: Defines the 10-field JSONL schema (id, title, status, priority, description, blocked_by, parent, created, updated, closed), SQLite cache schema, field constraints. References cli-command-structure-ux for status values, toon-output-format for output structure.
- **id-format-implementation**: Specifies ID generation - `tick-{6 hex chars}`, crypto/rand (stdlib), retry up to 5x for collisions, case-insensitive matching.
- **freshness-dual-write**: Implements storage synchronization - SHA256 hash-based freshness, JSONL-first with expendable SQLite cache, atomic rewrite (temp + fsync + rename), file locking via gofrs/flock.

**Coupling**: These three form the data foundation. The schema defines what's stored, ID format is part of that schema, and freshness/dual-write defines how data moves between JSONL and SQLite. Implementation would build these together as the storage engine.

### CLI Interface & Output
- **cli-command-structure-ux**: Defines all commands (create, start, done, cancel, reopen, list, show, dep add/rm, ready, blocked), TTY auto-detection for output format, flags (--toon, --pretty, --json, --quiet, --verbose), simple exit codes (0/1), error format.
- **toon-output-format**: Specifies TOON multi-section structure for complex data, format selection logic (inherits TTY detection from CLI discussion), empty arrays with zero count, plain text errors to stderr.
- **tui**: Defines human-readable output - raw fmt.Print (no TUI library), simple aligned columns, no colors/borders/interactivity.

**Coupling**: All three define how users interact with tick. The CLI discussion establishes TTY detection which TOON and TUI both build upon. TOON handles machine output, TUI handles human output. Together they specify the complete interface layer.

### Task Workflow & Validation
- **hierarchy-dependency-model**: Defines semantic rules - parent/child is organizational (not workflow constraint), blocked_by controls execution order, leaf-only `tick ready` rule (open + unblocked + no open children), child→parent deps disallowed (deadlock), parent→child allowed, cycles rejected at write time.
- **doctor-command-validation**: Specifies validation checks - 9 errors (cache staleness, JSONL syntax, duplicate IDs, ID format violations, orphaned references, self-refs, cycles, child→parent deadlock) + 1 warning (parent done while children open). Report-only behavior, separate `tick rebuild` command, human-readable output only.

**Coupling**: Doctor validates the rules established in hierarchy-dependency-model. The deadlock check (child blocked_by parent) comes directly from the hierarchy discussion. Orphan detection, cycle detection, and other validations enforce the workflow rules. These should be specified together.

## Independent Discussions

- **archive-strategy-implementation**: Deferred to post-v1. Decision: "No archiving - YAGNI." One file (tasks.jsonl), one cache (cache.db). Advisory system concept preserved for future use.

- **config-file-design**: Deferred to post-v1. Decision: "No config file - YAGNI." All defaults hardcoded (tick- prefix, priority 2). Customization via CLI flags only.

- **installation-options**: Distribution strategy independent of core functionality. Homebrew for macOS, install script for Linux/ephemeral environments, no Windows priority, no self-update. Can be specified separately.

- **migration-subcommand**: Import feature (`tick migrate --from beads`) with plugin/strategy pattern. One-time append, no sync/deduplication. Cleanly separable from core task management.

## Analysis Notes

**Cross-references identified:**
- data-schema-design → cli-command-structure-ux (status values)
- data-schema-design → toon-output-format (output structure)
- hierarchy-dependency-model → data-schema-design (blocked_by, parent fields)
- hierarchy-dependency-model → cli-command-structure-ux (tick ready behavior)
- doctor-command-validation → hierarchy-dependency-model (deadlock rule)
- doctor-command-validation → id-format-implementation (ID validation)
- toon-output-format → cli-command-structure-ux (TTY detection decided there)
- tui → toon-output-format (shares TTY detection)
- config-file-design → cli-command-structure-ux (deferred because flags handle it)

**Implementation order suggested:**
1. Core Data & Storage (foundation)
2. CLI Interface & Output (user interaction)
3. Task Workflow & Validation (business rules)
4. Independent features (distribution, migration)

**Deferred items (not for v1):**
- Archive strategy
- Config file

These explicitly concluded with "don't build it" and can be excluded from v1 specification entirely.
