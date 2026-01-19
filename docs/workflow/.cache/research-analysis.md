---
checksum: df242d3afab27dc11f98b42a4ec51352
generated: 2026-01-19
research_files:
  - exploration.md
---

# Research Analysis Cache

## Topics

### Data Schema Design
- **Source**: exploration.md (lines 62-138)
- **Summary**: Proposed JSONL format, task schema with 15+ fields, SQLite schema with tasks/dependencies tables, and ready_tasks view for deterministic querying.
- **Key questions**: Are all proposed fields necessary? Should any be added/removed? Is the SQLite schema optimal for expected queries?

### Freshness Check & Dual Write Implementation
- **Source**: exploration.md (lines 150-173)
- **Summary**: Cache freshness via hash/mtime comparison, dual write on mutations (JSONL + SQLite), atomic file writes using temp+rename pattern.
- **Key questions**: Hash vs mtime for freshness? How to handle partial write failures? Full file rewrite acceptable for updates?

### Agent Integration Patterns
- **Source**: exploration.md (lines 214-223)
- **Summary**: AGENTS.md concept defining how agents should interact with tick - never read raw files, always use --json, check ready first, update status when starting.
- **Key questions**: What instructions are essential? Should tick enforce any of these patterns? How to handle agent misbehavior?
- **Status**: No separate discussion needed

### Workflow Integration (claude-technical-workflows)
- **Source**: exploration.md (lines 273-302)
- **Summary**: Tick as new output format for planning phase, replacing/complementing existing formats (Local Markdown, Beads, Backlog.md, Linear). Planning agent creates tasks, implementation agent queries and executes.
- **Key questions**: How should the planning skill adapter work? What metadata beyond tasks is needed? How to handle plan versioning?
- **Status**: No separate discussion needed

### Distribution & Release Strategy
- **Source**: exploration.md (lines 545-553)
- **Summary**: Public GitHub repo, releases via goreleaser or manual, Homebrew via personal tap, dogfooding with claude-technical-workflows.
- **Key questions**: When is v1.0? What's the versioning strategy? How to handle breaking changes?
- **Discussed in**: installation-options.md (covers distribution methods, versioning, and updates)

### Doctor Command & Validation
- **Source**: exploration.md (lines 538-541)
- **Summary**: tick doctor for detecting cycles, orphaned tasks, cache corruption. Auto-rebuild from JSONL on corruption.
- **Key questions**: What validations are essential? Should doctor run automatically? How to report issues to agents vs humans?

### ID Format Implementation
- **Source**: exploration.md (lines 307-319)
- **Summary**: Hash-based IDs with customizable prefix ({prefix}-{hash}). Decision made but implementation details open.
- **Key questions**: Hash length (6-8 chars)? Random vs content-based? Collision handling?

### Hierarchy & Dependency Model
- **Source**: exploration.md (lines 324-358)
- **Summary**: Flat structure with parent field for hierarchy, blocked_by array for dependencies. Infinite depth, explicit completion.
- **Key questions**: Any edge cases in the model? How to handle bulk operations on hierarchies?

### Archive Strategy Implementation
- **Source**: exploration.md (lines 361-398)
- **Summary**: Keep done tasks in main file by default, optional tick archive command, --include-archived flag for searches.
- **Key questions**: Auto-archive threshold? Archive file indexing strategy? Unarchive command needed?

### Config File Design
- **Source**: exploration.md (lines 401-437)
- **Summary**: Flat key=value format in .tick/config, created at init, extensible with hardcoded defaults.
- **Key questions**: What initial config options? Environment variable overrides? Per-command config?

### CLI Command Structure & UX
- **Source**: exploration.md (lines 176-203)
- **Summary**: Core commands (init, create, list, show, start, done, reopen), aliases (ready, blocked), dependency commands, utility commands, global flags.
- **Key questions**: Command naming conventions? Flag consistency? Error message format?

### TOON Output Format
- **Source**: exploration.md (lines 473-511)
- **Summary**: TOON as default output for agent consumption (30-60% token savings), JSON via --json flag, --plain for human-readable.
- **Key questions**: TOON implementation details? Field ordering? Error output format?
