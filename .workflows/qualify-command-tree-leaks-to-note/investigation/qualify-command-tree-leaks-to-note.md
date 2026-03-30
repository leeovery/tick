# Investigation: Qualify Command Tree Leaks To Note

## Symptoms

### Problem Description

**Expected behavior:**
`tick note tree` should produce an error indicating "tree" is not a valid subcommand of "note". Flag validation should not reference "note tree" as if it were a real command.

**Actual behavior:**
`qualifyCommand` in `internal/cli/app.go` shares the `"tree"` case across both `"dep"` and `"note"` parent commands. This means `tick note tree` gets qualified as `"note tree"`, which isn't registered in `commandFlags`. When flags are passed (e.g., `tick note tree --foo`), the error says `unknown flag "--foo" for "note tree"`, implying "note tree" is a real command that just doesn't accept that flag.

### Manifestation

- Misleading error message: `unknown flag "--foo" for "note tree"` suggests the command exists
- `handleNote` eventually returns "unknown note sub-command 'tree'" without flags, which is correct but the flag validation path is wrong

### Reproduction Steps

1. Run `tick note tree --foo`
2. Observe error: `unknown flag "--foo" for "note tree"` (misleading)
3. Run `tick note tree` (without flags)
4. Observe error: `unknown note sub-command 'tree'` (correct but inconsistent path)

**Reproducibility:** Always

### Impact

- **Severity:** Low
- **Scope:** Unlikely user path
- **Business impact:** Poor UX for edge case — misleading error messages

### References

- Surfaced during dep-tree-visualization review
- `qualifyCommand` switch in `internal/cli/app.go`

---

## Analysis

### Initial Hypotheses

The `qualifyCommand` switch groups `"add"`, `"remove"`, and `"tree"` together for both `"dep"` and `"note"` parents. `"add"` and `"remove"` are valid for both, but `"tree"` only applies to `dep`.

### Code Trace

*To be filled during analysis*

### Root Cause

*To be filled during analysis*

### Contributing Factors

*To be filled during analysis*

### Why It Wasn't Caught

*To be filled during analysis*

### Blast Radius

*To be filled during analysis*

---

## Fix Direction

*To be filled after analysis*

---

## Notes

*To be filled as needed*
