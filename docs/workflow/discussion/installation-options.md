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

- [ ] What installation methods should we support?
      - Homebrew, direct binary, install script, go install?
      - Which are essential vs nice-to-have?
- [ ] How should the install script work?
      - What environments must it handle?
      - How to detect platform/architecture?
- [ ] Should we support Windows?
      - Is it a target environment?
      - What would be required?
- [ ] How do we handle versioning and updates?
      - Self-update capability?
      - Version pinning for agents?

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

*(Discussion in progress)*

