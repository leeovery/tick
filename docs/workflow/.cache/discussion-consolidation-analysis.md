---
checksum: 578ed29f8db05f82a23043afeadd3324
generated: 2026-01-21T18:20:00Z
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

### Core Data Storage
- **data-schema-design**: Defines JSONL/SQLite schemas, field constraints, task entity structure
- **freshness-dual-write**: Defines sync mechanism between JSONL and SQLite, atomic write patterns, hash-based freshness
- **id-format-implementation**: Defines ID format (tick-{6hex}), generation via crypto/rand, collision handling
- **hierarchy-dependency-model**: Defines how blocked_by and parent work together, leaf-only ready query, validation rules

**Coupling**: These are inseparable. Schema defines fields that ID format generates, freshness depends on schema structure, hierarchy/dependency model defines query behavior over that schema. All touch the same data structures.

**Note**: An existing `core-data-storage` specification exists and may already cover some of this content.

### CLI Commands & Output
- **cli-command-structure-ux**: Command structure, flags, error handling, TTY detection for format selection
- **toon-output-format**: TOON format specifics, multi-section approach for complex data, empty array handling
- **tui**: Human-readable output styling - simple aligned columns, no TUI library

**Coupling**: TOON and TUI formats are output modes controlled by CLI decisions. All three define what commands do and what they output.

### Diagnostics
- **doctor-command-validation**: 9 error checks + 1 warning, report-only behavior, human-readable output only

**Coupling**: References validation rules from hierarchy-dependency-model and id-format-implementation. Could stand alone or merge into CLI spec.

### Migration
- **migration-subcommand**: Plugin architecture for importing from other tools, beads as initial provider

**Coupling**: Standalone utility. References task schema but otherwise independent.

### Installation
- **installation-options**: Homebrew (macOS), install script (Linux/ephemeral), global installation

**Coupling**: Standalone topic about distribution. No coupling to other discussions.

## Deferred Decisions (No Specification Needed)
- **archive-strategy-implementation**: Concluded with "defer archiving entirely - not needed for v1"
- **config-file-design**: Concluded with "no config file for v1"

These discussions reached YAGNI conclusions - no implementation required, thus no specification needed.

## Analysis Notes

The 12 discussions naturally cluster into:
1. **Core data layer** (4) - tightly coupled, should be one specification
2. **CLI interface** (3) - tightly coupled, should be one specification
3. **Utilities** (2) - doctor and migration can stand alone
4. **Distribution** (1) - installation is independent
5. **YAGNI** (2) - no specs needed, just documented decisions

The existing core-data-storage specification should be reviewed - it may already contain content from the core data layer discussions. If so, this grouping would "continue" that spec rather than create a new one.
