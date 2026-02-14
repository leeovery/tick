---
topic: installation
status: concluded
format: local-markdown
specification: ../specification/installation.md
spec_commit: d20af03c956b3e3c4608eb4d5945a67c5a3f6c90
created: 2026-01-31
updated: 2026-01-31
external_dependencies:
  - topic: tick-core
    description: Buildable Go binary that compiles successfully
    state: resolved
    task_id: tick-core-1-5
planning:
  phase: 3
  task: 4
---

# Plan: Installation

## Overview

**Goal**: Distribute tick as a cross-platform Go CLI via install script, Homebrew, and GitHub Releases.

**Done when**:
- Pushing a version tag produces release assets for all four platforms
- Install script works on Linux (download binary) and macOS (delegate to Homebrew)
- Homebrew tap and formula provide native macOS installation

**Key Decisions** (from specification):
- Global installation model (not per-project vendoring)
- macOS install script delegates to Homebrew rather than direct binary download (avoids code signing complexity)
- No self-update capability — updates via original install method
- No Windows automated install path for v1

## Phases

### Phase 1: Release Pipeline and Install Script
status: approved
approved_at: 2026-01-31

**Goal**: Establish a minimal buildable Go binary, goreleaser configuration, GitHub Actions release workflow, and Linux install script so that pushing a tag produces release assets and the install script can download and install the binary on Linux.

**Why this order**: This is the walking skeleton — the thinnest end-to-end slice proving the entire distribution pipeline from source to installed binary. goreleaser needs something to build, the install script needs assets to download, and the GitHub Actions workflow ties it together. All subsequent distribution methods depend on this foundation existing.

**Acceptance**:
- [ ] A minimal Go main package exists and compiles successfully
- [ ] goreleaser configuration produces archives matching the naming convention (`tick_X.Y.Z_{os}_{arch}.tar.gz`) for all four platforms (darwin-amd64, darwin-arm64, linux-amd64, linux-arm64)
- [ ] GitHub Actions workflow triggers goreleaser on version tag push
- [ ] `scripts/install.sh` detects Linux OS and architecture (`x86_64`->`amd64`, `aarch64`->`arm64`, `arm64`->`arm64`)
- [ ] `scripts/install.sh` downloads the correct release asset and installs to `/usr/local/bin` or `~/.local/bin` fallback
- [ ] `scripts/install.sh` is idempotent (overwrites existing binary without error)

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| installation-1-1 | Minimal Go Binary | none | completed |
| installation-1-2 | goreleaser Configuration | archive naming must match spec convention exactly | completed |
| installation-1-3 | GitHub Actions Release Workflow | workflow should only trigger on semver tags | completed |
| installation-1-4 | Linux Install Script | `/usr/local/bin` not writable triggers `~/.local/bin` fallback, `~/.local/bin` may not exist, overwrite existing binary, unsupported architecture | completed |

---

### Phase 2: Homebrew Distribution and macOS Install Path
status: approved
approved_at: 2026-01-31

**Goal**: Add Homebrew tap and formula for macOS distribution, implement macOS behavior in the install script, and harden error handling across all install paths.

**Why this order**: Phase 1 established the release pipeline and Linux install path. This phase completes macOS support (the other primary audience) and hardens edge cases. It depends on release assets being available from Phase 1's goreleaser setup. The Homebrew formula references the same release asset naming convention, and the macOS install script path delegates to Homebrew — both require the Phase 1 infrastructure to exist.

**Acceptance**:
- [ ] Homebrew tap repository structure exists with a working formula that installs tick from GitHub release assets
- [ ] `scripts/install.sh` on macOS delegates to Homebrew when `brew` is available (runs `brew tap {owner}/tick && brew install tick`)
- [ ] `scripts/install.sh` on macOS without Homebrew exits with instructive error message directing user to install via Homebrew
- [ ] `scripts/install.sh` fails cleanly with meaningful error when download fails, unsupported OS is detected, or unsupported architecture is encountered
- [ ] Install script works correctly when run via `curl -fsSL ... | bash` (no interactive prompts, correct exit codes)

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| installation-2-1 | Homebrew Tap Repository and Formula | formula must handle both Intel and Apple Silicon macOS, version in formula URL must strip leading v | completed |
| installation-2-2 | macOS Install Script: Homebrew Delegation | brew tap or brew install failure should propagate exit code, tick already installed via Homebrew (idempotent re-install) | completed |
| installation-2-3 | macOS Install Script: No Homebrew Error Path | none | completed |
| installation-2-4 | Install Script Error Handling Hardening | script piped via curl with server error, partial download, OS value that is neither Linux nor Darwin (e.g. FreeBSD) | completed |

---

### Phase 3: Analysis (cycle 1)
status: approved

**Goal**: Address findings from implementation analysis cycle 1.

#### Tasks
| ID | Name | Edge Cases | Status |
|----|------|------------|--------|
| installation-3-1 | Extract shared findRepoRoot test utility | — | authored |
| installation-3-2 | Extract step-search helper in release_test.go | — | authored |
| installation-3-3 | Document Homebrew tap repository requirement | — | authored |
| installation-3-4 | Add cross-component asset naming contract test | — | authored |

---

## Log

| Date | Change |
|------|--------|
| 2026-01-31 | Created from specification |
| 2026-01-31 | Plan concluded — 2 phases, 8 tasks |
| 2026-02-14 | Phase 3 added — 4 analysis tasks from cycle 1 |
