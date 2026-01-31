---
id: installation-1-3
phase: 1
status: pending
created: 2026-01-31
---

# GitHub Actions Release Workflow

## Goal

goreleaser (configured in installation-1-2) needs a CI trigger to run automatically when a new version is released. Without a GitHub Actions workflow, releases would require manual invocation of goreleaser locally, which is error-prone, environment-dependent, and not reproducible. This task creates a GitHub Actions workflow that triggers goreleaser exclusively on semantic version tag pushes, producing release assets for all four platforms automatically.

## Implementation

1. **Create the workflow file** at `.github/workflows/release.yml`.

2. **Configure the trigger** to fire only on semver tags:
   ```yaml
   on:
     push:
       tags:
         - 'v[0-9]+.[0-9]+.[0-9]+'
   ```
   This pattern matches `v1.0.0`, `v0.1.0`, `v12.34.56`, etc. It does NOT match `v1.0.0-beta`, `v1.0.0-rc.1`, `latest`, `nightly`, or tags missing the `v` prefix.

3. **Define permissions** — the workflow needs `contents: write` to create GitHub releases and upload assets:
   ```yaml
   permissions:
     contents: write
   ```

4. **Define the release job** with the following steps:

   a. **Checkout** the repository with full history (goreleaser needs tags for version detection):
      ```yaml
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      ```

   b. **Set up Go** using the version from `go.mod`:
      ```yaml
      - uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'
      ```

   c. **Run goreleaser** using the official action:
      ```yaml
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      ```

5. **Set the runner** to `ubuntu-latest`. goreleaser handles cross-compilation for all target platforms from a single Linux runner via Go's cross-compile support — no matrix build needed.

6. **Name the workflow** clearly, e.g., `name: Release`.

## Tests

- `"workflow triggers on valid semver tag v1.0.0"` — parse the workflow YAML and confirm the tag pattern matches `v1.0.0`
- `"workflow triggers on valid semver tag v0.1.0"` — confirm the tag pattern matches `v0.1.0`
- `"workflow triggers on valid semver tag v12.34.56"` — confirm the tag pattern matches multi-digit versions
- `"workflow does not trigger on pre-release tag v1.0.0-beta"` — confirm the tag pattern does NOT match `v1.0.0-beta`
- `"workflow does not trigger on pre-release tag v1.0.0-rc.1"` — confirm the tag pattern does NOT match `v1.0.0-rc.1`
- `"workflow does not trigger on non-version tag latest"` — confirm the tag pattern does NOT match `latest`
- `"workflow does not trigger on branch push to main"` — confirm no branch trigger is configured
- `"workflow does not trigger on tag without v prefix like 1.0.0"` — confirm the tag pattern requires the `v` prefix
- `"checkout step uses fetch-depth 0 for full history"` — parse the workflow YAML and confirm `fetch-depth: 0` is set on the checkout step
- `"goreleaser step has GITHUB_TOKEN configured"` — confirm the environment variable is passed to the goreleaser step
- `"workflow has contents write permission"` — confirm `permissions.contents` is set to `write`

## Edge Cases

- **Semver-only triggering**: The tag pattern `v[0-9]+.[0-9]+.[0-9]+` must NOT match pre-release suffixes (e.g., `v1.0.0-beta`, `v1.0.0-rc.1`), build metadata (e.g., `v1.0.0+build.123`), non-version tags (e.g., `latest`, `nightly`), or tags missing the `v` prefix (e.g., `1.0.0`). GitHub Actions tag patterns use glob matching, not full regex — the pattern anchors to the full ref name and does not match partial strings with suffixes.
- **Shallow clone breaks goreleaser**: goreleaser uses `git describe --tags` to determine the version. If `fetch-depth` is not set to `0`, the checkout will be shallow and goreleaser will fail or produce an incorrect version string. The `fetch-depth: 0` setting on the checkout step is essential.
- **Missing GITHUB_TOKEN**: Without the `GITHUB_TOKEN` environment variable, goreleaser cannot create the GitHub release or upload assets. The token is automatically provided by GitHub Actions via `secrets.GITHUB_TOKEN` — no manual secret configuration is required.

## Acceptance Criteria

- [ ] `.github/workflows/release.yml` exists and is valid YAML
- [ ] Workflow triggers only on tag pushes matching the semver pattern `v[0-9]+.[0-9]+.[0-9]+`
- [ ] No branch push triggers or pull request triggers are configured
- [ ] Workflow has `permissions: contents: write`
- [ ] Checkout step uses `fetch-depth: 0` for full git history
- [ ] Go is set up using `go-version-file: 'go.mod'`
- [ ] goreleaser is invoked with `release --clean` and `GITHUB_TOKEN` is passed as an environment variable
- [ ] Workflow runs on `ubuntu-latest`

## Context

The specification defines four platform targets for release assets: `darwin-amd64`, `darwin-arm64`, `linux-amd64`, and `linux-arm64`. goreleaser (configured in installation-1-2) handles the cross-compilation and archive creation. This workflow's sole responsibility is to trigger goreleaser at the right time (semver tag push) with the right environment (full git history, Go toolchain, GitHub token).

The specification states that release assets follow the naming convention `tick_X.Y.Z_{os}_{arch}.tar.gz`. goreleaser derives the version from the git tag — the `v` prefix is stripped automatically by goreleaser (e.g., tag `v1.0.0` produces `tick_1.0.0_...`). This means the tag format directly influences asset naming, reinforcing why the trigger pattern must be strict.

The install script (installation-1-4) downloads assets from GitHub Releases, so the workflow must successfully create a release with uploaded assets. The Homebrew formula (Phase 2) also depends on assets being available at predictable URLs under GitHub Releases.

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
