---
id: migration-2-5
phase: 2
status: completed
created: 2026-01-31
---

# Unknown Provider Available Listing

## Goal

Phase 1 (migration-1-5) established a provider registry with `NewProvider(name string) (Provider, error)` that returns a minimal error `unknown provider "<name>"` when the name is not recognized. The specification requires more: "If `--from` specifies an unrecognized provider, exit immediately with an error listing available providers." The current error tells the user what went wrong but not what options exist. This task enhances the registry to produce a structured error message that includes the list of all registered providers, matching the spec's exact format:

```
Error: Unknown provider "xyz"

Available providers:
  - beads
```

Although only "beads" exists today, the implementation must work correctly for any number of registered providers — the format must scale as new providers are added.

## Implementation

- Add a function to the registry in `internal/migrate/` that returns the list of registered provider names:
  ```go
  // e.g., internal/migrate/registry.go
  func AvailableProviders() []string
  ```
  This returns a sorted slice of all provider names registered in the map (e.g., `["beads"]`). Sorting ensures deterministic output for tests and consistent user experience regardless of map iteration order.

- Create a custom error type (or a formatted error) for the unknown provider case that includes the available providers list. One approach:
  ```go
  type UnknownProviderError struct {
      Name       string
      Available  []string
  }

  func (e *UnknownProviderError) Error() string {
      // Produces:
      // Unknown provider "xyz"
      //
      // Available providers:
      //   - beads
  }
  ```
  The `Error()` method formats the full multi-line message. Each available provider is listed on its own line with a two-space indent and a dash prefix (`  - beads`). An empty line separates the error line from the "Available providers:" header.

- Modify `NewProvider` in `internal/migrate/registry.go` to return an `*UnknownProviderError` (instead of the plain `fmt.Errorf` from Phase 1) when the name is not found. The error includes the result of `AvailableProviders()`.

- In the `migrate` CLI command handler (established in migration-1-5), the unknown provider error is already printed to stderr and exits with code 1. No changes needed in the CLI handler — the error's `Error()` method now produces the enhanced message, and the existing error printing will output the full formatted string. Verify the CLI prints `Error: ` as a prefix before the error message to match the spec's format:
  ```
  Error: Unknown provider "xyz"

  Available providers:
    - beads
  ```
  If the CLI currently wraps the error differently, adjust the `UnknownProviderError.Error()` output so the final stderr output matches the spec exactly.

- The `UnknownProviderError` should be exported so the CLI (or tests) can type-assert on it if needed (e.g., `errors.As(err, &unknownErr)`). This also enables future callers to programmatically inspect the available providers list.

- Ensure alphabetical sort of provider names in the available list. With only "beads" today, sorting is trivial, but the implementation should use `sort.Strings` to be future-proof.

## Tests

- `"NewProvider returns UnknownProviderError for unrecognized name"`
- `"UnknownProviderError.Error includes the unknown provider name in quotes"`
- `"UnknownProviderError.Error includes 'Available providers:' header"`
- `"UnknownProviderError.Error lists each registered provider with '  - ' prefix"`
- `"AvailableProviders returns sorted list of registered provider names"`
- `"single provider in registry: error message lists one provider"`
- `"multiple providers in registry: error message lists all providers alphabetically"`
- `"UnknownProviderError includes empty line between error line and available list"`
- `"CLI prints 'Error: Unknown provider \"xyz\"' followed by available providers to stderr"`
- `"NewProvider still returns BeadsProvider for name 'beads'"` (regression)
- `"UnknownProviderError is type-assertable via errors.As"`

## Edge Cases

**Single provider in registry**: Only "beads" is registered (the current state). The available providers list contains exactly one entry. The output format is:
```
Unknown provider "xyz"

Available providers:
  - beads
```
Tests must verify this works correctly — it is the real-world case for the foreseeable future.

**Multiple providers in registry**: When additional providers are registered in the future (e.g., "jira", "linear"), the list shows all of them alphabetically:
```
Unknown provider "xyz"

Available providers:
  - beads
  - jira
  - linear
```
To test this edge case, temporarily register additional mock providers in the test (add entries to the registry map, then clean up after). This verifies the format scales correctly without requiring real provider implementations. The test should confirm alphabetical ordering and that each provider appears on its own line with the correct prefix.

## Acceptance Criteria

- [ ] `NewProvider` returns an `*UnknownProviderError` (not a plain error) for unrecognized names
- [ ] `UnknownProviderError.Error()` produces the spec-mandated multi-line format with provider name in quotes
- [ ] Available providers are listed alphabetically, each on its own line with `  - ` prefix
- [ ] An empty line separates the error line from the "Available providers:" section
- [ ] `AvailableProviders()` function exists and returns a sorted slice of registered provider names
- [ ] The CLI stderr output matches the spec format: `Error: Unknown provider "<name>"` followed by the available list
- [ ] `NewProvider("beads")` still returns a valid `BeadsProvider` (regression)
- [ ] Single-provider listing is tested and correct
- [ ] Multiple-provider listing is tested (via test-scoped registry entries) and alphabetically sorted
- [ ] `UnknownProviderError` is exported and supports `errors.As` type assertion
- [ ] All tests written and passing

## Context

The specification is explicit about the unknown provider error format:

> **Unknown provider**: If `--from` specifies an unrecognized provider, exit immediately with an error listing available providers:
> ```
> Error: Unknown provider "xyz"
>
> Available providers:
>   - beads
> ```

Phase 1 (migration-1-5) intentionally deferred this to Phase 2: "Phase 1 uses a minimal error message; Phase 2 will enhance this to list available providers." The registry's `NewProvider` function and the CLI's error handling path already exist — this task upgrades the error content without changing the control flow.

The provider registry in `internal/migrate/registry.go` is a hardcoded `map[string]` factory. The `AvailableProviders()` function iterates this map's keys. Since Go map iteration order is non-deterministic, the function must sort the keys before returning to ensure consistent output.

Specification reference: `docs/workflow/specification/migration.md`
