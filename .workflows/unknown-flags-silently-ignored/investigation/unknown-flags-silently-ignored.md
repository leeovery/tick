# Investigation: Unknown Flags Silently Ignored

## Symptoms

### Problem Description

**Expected behavior:**
All commands should reject unrecognised flags with a clear error message, e.g. `Error: unknown flag --blocks. Run 'tick help dep' for usage.`

**Actual behavior:**
Unknown flags are silently discarded. Any argument starting with `-` that isn't a known flag is stripped without warning, which can mislead users into thinking a flag had effect when it didn't.

### Manifestation

- No error messages produced when unknown flags are passed
- Commands execute with the unknown flag silently dropped
- User intent can be misinterpreted — e.g. `tick dep add tick-aaa --blocks tick-bbb` silently ignores `--blocks` (only valid on `create`/`update`) and treats it as `tick dep add tick-aaa tick-bbb`

### Reproduction Steps

1. Run any tick command with an unknown flag, e.g. `tick dep add tick-aaa --blocks tick-bbb`
2. Observe: command succeeds without error
3. The `--blocks` flag was silently ignored; the command behaved as if it was `tick dep add tick-aaa tick-bbb`

**Reproducibility:** Always

### Impact

- **Severity:** Low
- **Scope:** All commands affected
- **Business impact:** User confusion, potential for unintended task relationships

---

## Analysis

### Initial Hypotheses

Flag parsing across commands likely uses a pattern that strips unknown flags rather than rejecting them. Need to trace how flags are parsed in the CLI layer.

### Code Trace

*To be completed during analysis*

### Root Cause

*To be determined*

### Contributing Factors

*To be determined*

### Why It Wasn't Caught

*To be determined*

### Blast Radius

*To be determined*

---

## Fix Direction

*To be determined after analysis*

---

## Notes

- Referenced from `bugs.md` as BUG-1
- General CLI parsing concern — not specific to any single command
