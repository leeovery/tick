TASK: Document Homebrew tap repository requirement (installation-3-3)

ACCEPTANCE CRITERIA:
- homebrew-tap/README.md clearly states the formula must be published to a separate homebrew-tick repository
- The sync requirement is documented as a deployment prerequisite

STATUS: Issues Found

SPEC CONTEXT: The specification says `brew tap {owner}/tick && brew install tick`, which requires a GitHub repo at `github.com/leeovery/homebrew-tick`. The original analysis (cycle 1) identified that the formula living inside the main tick repo under `homebrew-tap/` would cause `brew tap` discovery to fail at runtime. The task proposed documenting this gap in `homebrew-tap/README.md`. Since the analysis was written, the project underwent a significant migration (commit c4a2a84): the entire `homebrew-tap/` directory was removed from this repo, and the formula was moved to a shared `leeovery/homebrew-tools` repository. The release workflow now dispatches formula updates automatically via `repository_dispatch`.

IMPLEMENTATION:
- Status: Drifted (superseded by migration)
- Location: No `homebrew-tap/README.md` exists. The entire `homebrew-tap/` directory was removed.
- Notes:
  - The underlying problem this task identified (gap between in-repo formula and runtime `brew tap` expectations) was solved more thoroughly than planned: instead of documenting the gap, the formula was moved entirely out of this repo.
  - The release workflow at `.github/workflows/release.yml` (lines 44-58) handles the integration automatically via `repository_dispatch` to `leeovery/homebrew-tools`, making manual sync documentation unnecessary.
  - The install script at `/Users/leeovery/Code/tick/scripts/install.sh` (line 66) correctly references `leeovery/tools/tick`, consistent with the migration.
  - The project's `CLAUDE.md` (line 59) documents: "formula lives in separate `homebrew-tools` repo, updated via GitHub Actions `repository_dispatch`". This partially fulfills the documentation intent, though in a different location and form than planned.
  - The specific deliverables (homebrew-tap/README.md with deployment prerequisite documentation) do not exist, but the need for them has been eliminated by the migration to an automated workflow.

TESTS:
- Status: N/A
- Coverage: This is a documentation task. No automated tests were required per the task specification.
- Notes: No tests needed.

CODE QUALITY:
- Project conventions: N/A (documentation task)
- SOLID principles: N/A
- Complexity: N/A
- Modern idioms: N/A
- Readability: N/A
- Issues: None

BLOCKING ISSUES:
- None. The task's acceptance criteria are technically unmet (no `homebrew-tap/README.md` exists), but the underlying problem the task was designed to solve has been resolved more completely by the migration to `homebrew-tools` with automated `repository_dispatch`. The acceptance criteria are obsolete rather than unmet.

NON-BLOCKING NOTES:
- The task status in the plan should ideally be updated from "completed" to something that reflects the migration superseded it (e.g., "superseded" or a note explaining the migration resolved the underlying concern). The task was never actually implemented as specified -- the migration happened instead.
- The CLAUDE.md line 59 serves as the de facto documentation of the external Homebrew dependency, but it is terse. Consider whether a more detailed note about the `repository_dispatch` integration (what payload fields are expected, what the `homebrew-tools` repo does with them) would be valuable for onboarding.
- The plan task `installation-3-3.md` still references `homebrew-tick` and `homebrew-tap/` throughout, which no longer exist. These references are stale.
