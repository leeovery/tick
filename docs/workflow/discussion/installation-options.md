# Discussion: Installation Options

**Date**: 2026-01-19
**Status**: Exploring

## Context

Tick is a Go CLI tool that needs to be installable across various environments. The research phase proposed: GitHub releases (via goreleaser or manual), Homebrew via personal tap. However, this leaves gaps.

**Problem**: Different environments have different installation capabilities:
- **macOS with Homebrew**: Well-served by the proposed approach
- **Linux**: Homebrew exists but isn't universally installed; many prefer native package managers or direct binaries
- **Claude Code for Web**: Containerized environment without Homebrew or typical package managers
- **Windows**: Status unclear - is it even a target?

**Key constraint**: Tick is agent-first. Agents running in constrained environments (like Claude Code for Web) need a reliable way to install tick without human intervention or complex setup.

### References

- Research: exploration.md (lines 545-553) - Distribution section
- Go compiles to static binaries - simplifies distribution

## Questions

- [x] Global or per-project installation?
- [x] What installation methods should we support?
      - Homebrew, direct binary, install script, go install?
      - Which are essential vs nice-to-have?
- [x] How should the install script work?
      - What environments must it handle?
      - How to detect platform/architecture?
      - Ephemeral environment considerations?
- [ ] Should we support Windows?
      - Is it a target environment?
      - What would be required?
- [ ] How do we handle versioning and updates?
      - Self-update capability?
      - Version pinning for agents?

---

## Global or per-project installation?

### Context
Should tick be installed once on a user's system, or vendored into each project?

### Options Considered

**Global installation**
- Installed to user's system (e.g., `/usr/local/bin`, `~/.local/bin`)
- Single version shared across projects
- Respects user's home system conventions

**Per-project (vendored)**
- Binary stored in repository
- Each project has its own version
- No system-wide installation needed

### Journey

Initial thought was to consider per-project for reproducibility. But:
- Adds significant size to repositories (Go binaries are ~10-20MB)
- Version drift across projects
- Awkward for a CLI tool - not a library
- No clear benefit over global install

**Ephemeral environments** (like Claude Code for Web) are the interesting case: there's no persistent "global" install. Each session starts fresh. The solution: install script runs as part of session setup, installs to user-writable location, cached if environment supports it.

### Decision

**Global installation**. Per-project vendoring adds bloat without clear benefit. For ephemeral environments, install script handles session setup.

---

## What installation methods should we support?

### Context
Go produces static binaries, which simplifies distribution. But users need a way to get those binaries onto their systems. Different environments have different capabilities.

### Options Considered

**Homebrew (macOS/Linux)**
- Pros: Familiar to developers, handles updates, existing personal tap available
- Cons: Requires Homebrew installed, not available in all environments

**GitHub Releases with pre-built binaries**
- Pros: Universal, no dependencies, goreleaser automates multi-platform builds
- Cons: Manual download/install, no automatic updates

**Install script (curl | bash pattern)**
- Pros: Works anywhere with curl/bash, can detect platform, familiar pattern
- Cons: Security concerns (though common), requires maintaining the script

**go install**
- Pros: Native to Go ecosystem, simple
- Cons: Requires Go toolchain installed

### Journey

Research proposed Homebrew + GitHub releases. But testing in Claude Code for Web revealed: no Homebrew available, but curl and Go toolchain present.

**Emerging priority**:
1. **Install script** - essential for constrained environments, works everywhere with curl/bash
2. **GitHub releases** - required anyway (install script pulls from here)
3. **Homebrew** - nice-to-have for better UX on systems that have it
4. **go install** - free if repo structured correctly, but requires Go toolchain

**Key insight**: Install script may need to be the *primary* documented method, not Homebrew. Homebrew is an optimization for those who have it.

For ephemeral environments: install script runs at session start, needs to be fast and idempotent.

### Decision

Support all four methods, prioritized:
1. **Install script** - primary method, documented first
2. **Homebrew** - alternative for those who prefer it (via personal tap)
3. **GitHub releases** - manual download option
4. **go install** - for Go developers

---

## How should the install script work?

### Context

The install script is the primary installation method. Needs to work across macOS and Linux, be fast for ephemeral environments, and simple to maintain.

### Options Considered

**Install location**
- `/usr/local/bin` - traditional, but may need sudo
- `~/.local/bin` - user-writable, XDG-compliant
- Custom `~/.tick/bin` - isolated but requires PATH modification

**Version handling**
- Always download latest
- Check version and update if outdated
- Skip entirely if already installed

**Fallback strategy**
- Binary only (fail if unavailable)
- Binary → go install → source build (like Beads)

### Journey

Researched how [Beads](https://github.com/steveyegge/beads) handles this. Their install script:
- Tries `/usr/local/bin` first (if writable), falls back to `~/.local/bin`
- Detects platform via `uname -s` (OS) and `uname -m` (arch)
- Has multiple fallbacks: binary download → go install → source build

For tick, we want **simpler**:
- Use conventional locations: `/usr/local/bin` if writable, else `~/.local/bin`
- Platform detection same as Beads (proven pattern)
- **No update logic**: download latest, skip if already installed
- Keep it simple - updates are out of scope for the script

The "skip if installed" behavior is important for ephemeral environments where the script runs every session. If tick is already there (perhaps cached), don't waste time re-downloading.

### Decision

**Install location**: `/usr/local/bin` if writable, else `~/.local/bin` (follow Beads pattern)

**Behavior**:
- Download latest binary from GitHub releases
- Skip if `tick` already exists in PATH
- No update/upgrade functionality - keep script simple

**Platform support**: macOS (darwin) and Linux, amd64 and arm64

