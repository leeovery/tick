---
id: installation-1-2
phase: 1
status: completed
created: 2026-01-31
---

# goreleaser Configuration

## Goal

The release pipeline needs goreleaser to produce platform-specific archives from the Go binary created in task installation-1-1. Without this configuration, there is no automated way to build, package, and name release assets consistently. This task creates the `.goreleaser.yaml` file that produces archives for all four target platforms using the exact naming convention defined in the specification (`tick_X.Y.Z_{os}_{arch}.tar.gz`). The install script (task installation-1-4) and Homebrew formula (Phase 2) both depend on these archive names being predictable and correct.

## Implementation

1. **Create `.goreleaser.yaml`** at the repository root with the following configuration:

   - **project_name**: `tick`
   - **builds**: A single build entry targeting the main package at `./cmd/tick/`:
     - `id: tick`
     - `binary: tick`
     - `main: ./cmd/tick/`
     - `env: [CGO_ENABLED=0]` (static binaries, no C dependencies)
     - `goos: [darwin, linux]`
     - `goarch: [amd64, arm64]`
     - `ldflags`: `-s -w` (strip debug info for smaller binaries)
   - **archives**: A single archive entry:
     - `id: tick`
     - `format: tar.gz`
     - `name_template: "tick_{{ .Version }}_{{ .Os }}_{{ .Arch }}"` — this produces the exact naming convention from the specification: `tick_X.Y.Z_{os}_{arch}.tar.gz`
     - `files: [none*]` — include only the binary, no extra files. Use a glob pattern that matches nothing extra so the archive contains just the `tick` binary.
   - **changelog**: `disable: true` (or `skip: true` depending on goreleaser version) — changelog generation is not required for this project.
   - **release**: Configure for GitHub releases (the default; ensure `draft: false`).

2. **Validate the configuration** by running goreleaser in check mode:
   - Run `goreleaser check` to validate the YAML syntax and configuration structure.

3. **Test a local snapshot build** to verify archive naming:
   - Run `goreleaser build --snapshot --clean` to produce binaries without publishing.
   - Verify that the build produces binaries for all four platform combinations: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`.

4. **Verify archive naming with a full local release dry run**:
   - Run `goreleaser release --snapshot --clean --skip=publish` to produce archives locally without publishing.
   - Verify the `dist/` directory contains archives matching the naming pattern.
   - Note: snapshot builds use a synthetic version string; the naming template itself is what matters. When triggered by a real git tag (e.g., `v1.0.0`), `{{ .Version }}` resolves to `1.0.0`, producing `tick_1.0.0_darwin_amd64.tar.gz` etc.

5. **Verify each archive contains only the `tick` binary**:
   - Extract one of the snapshot archives and confirm it contains only the `tick` binary.

6. **Add `dist/` to `.gitignore`**:
   - goreleaser outputs build artifacts to `dist/`. Ensure this directory is excluded from version control.

## Tests

- `"goreleaser check passes without errors"` — run `goreleaser check` and assert exit code 0
- `"snapshot build produces binaries for all four platforms"` — run `goreleaser build --snapshot --clean` and assert the dist directory contains build outputs for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- `"snapshot release produces correctly named tar.gz archives"` — run `goreleaser release --snapshot --clean --skip=publish` and assert four `.tar.gz` files exist in `dist/` with names matching the pattern `tick_*_darwin_amd64.tar.gz`, `tick_*_darwin_arm64.tar.gz`, `tick_*_linux_amd64.tar.gz`, `tick_*_linux_arm64.tar.gz`
- `"archive contains only the tick binary"` — extract a snapshot archive and assert the only file is `tick` (no README, LICENSE, or other files)
- `"archive name_template uses correct convention"` — inspect `.goreleaser.yaml` and assert `name_template` is `tick_{{ .Version }}_{{ .Os }}_{{ .Arch }}` (matches specification pattern `tick_X.Y.Z_{os}_{arch}.tar.gz`)

## Edge Cases

**Archive naming must match spec convention exactly**: The specification defines the asset naming as `tick_X.Y.Z_{os}_{arch}.tar.gz`. goreleaser's default `name_template` is `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}` which would produce the same result *if* `project_name` is `tick`. However, goreleaser's defaults have changed across versions and may include additional template variables. The `name_template` must be set explicitly to `tick_{{ .Version }}_{{ .Os }}_{{ .Arch }}` rather than relying on defaults. This is critical because the install script (task installation-1-4) constructs download URLs using this exact pattern — any deviation will break automated installation.

Additionally, verify that goreleaser does not append format-specific suffixes beyond `.tar.gz`. The archive format is set to `tar.gz` explicitly to prevent goreleaser from choosing a platform-specific default (e.g., `.zip` for darwin).

## Acceptance Criteria

- [ ] `.goreleaser.yaml` exists at the repository root
- [ ] Configuration targets exactly four platform combinations: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- [ ] Build points to `./cmd/tick/` as the main package with binary name `tick`
- [ ] Archive `name_template` is explicitly set to `tick_{{ .Version }}_{{ .Os }}_{{ .Arch }}`
- [ ] Archive format is explicitly set to `tar.gz` (not relying on defaults)
- [ ] `goreleaser check` passes without errors
- [ ] Snapshot release produces four `.tar.gz` archives matching the naming pattern
- [ ] Each archive contains only the `tick` binary (no extra files)
- [ ] `dist/` is listed in `.gitignore`

## Context

The specification defines the release asset naming convention as:

> goreleaser convention: `{binary}_{version}_{os}_{arch}.tar.gz`
>
> Assets per release:
> - `tick_X.Y.Z_darwin_amd64.tar.gz`
> - `tick_X.Y.Z_darwin_arm64.tar.gz`
> - `tick_X.Y.Z_linux_amd64.tar.gz`
> - `tick_X.Y.Z_linux_arm64.tar.gz`
>
> Each archive contains the `tick` binary.

This naming convention is load-bearing infrastructure — the install script constructs download URLs from it, and the Homebrew formula references it for asset URLs. Any deviation from this pattern breaks downstream consumers.

Windows is intentionally excluded from `goos` to keep the configuration aligned with the four explicitly listed platforms per spec. The specification states: "goreleaser configuration can be set up early with placeholder build targets" — this task builds on the minimal binary from task installation-1-1.

Specification reference: `docs/workflow/specification/installation.md` (for ambiguity resolution)
