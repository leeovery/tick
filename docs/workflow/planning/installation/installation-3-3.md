---
id: installation-3-3
phase: 3
status: completed
created: 2026-02-14
---

# Document Homebrew tap repository requirement

**Problem**: The spec says `brew tap {owner}/tick` which requires a GitHub repo at `github.com/leeovery/homebrew-tick`. The current formula lives inside the main tick repo under `homebrew-tap/`. The `brew tap leeovery/tick` command looks for `github.com/leeovery/homebrew-tick`, not `github.com/leeovery/tick/homebrew-tap/`, so tap discovery will fail at runtime. The install script (line 66) also runs `brew tap leeovery/tick`.

**Solution**: Add clear documentation in homebrew-tap/README.md explaining that the formula files must be copied or synced to a separate `homebrew-tick` repository for `brew tap` to work. This documents the deployment prerequisite without changing the development structure.

**Outcome**: Any developer or CI process knows that the formula directory contents must be published to a separate homebrew-tick repo. The gap between development layout and runtime expectation is explicitly documented.

**Do**:
1. Create or update `homebrew-tap/README.md` to explain that the formula lives here for development but must be synced to a separate `github.com/leeovery/homebrew-tick` repository for `brew tap leeovery/tick` to work
2. Document that the release workflow or a manual step is responsible for this sync
3. Note that `scripts/install.sh` line 66 depends on this external repo existing

**Acceptance Criteria**:
- homebrew-tap/README.md clearly states the formula must be published to a separate homebrew-tick repository
- The sync requirement is documented as a deployment prerequisite

**Tests**:
- No automated tests required; this is a documentation task
