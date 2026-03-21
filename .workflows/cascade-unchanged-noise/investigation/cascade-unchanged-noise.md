# Investigation: Cascade Unchanged Noise

## Symptoms

### Problem Description

**Expected behavior:**
When a status transition triggers cascades, only the primary transition and genuinely cascaded changes appear in the output.

**Actual behavior:**
Output includes "(unchanged)" lines for sibling and descendant tasks that were already in a terminal state and didn't actually change. These add visual noise that makes it harder to see what actually happened.

### Manifestation

Noisy CLI output. Example:

```
$ tick done tick-b15fda
tick-b15fda: in_progress → done
tick-c5a1ff: in_progress → done (auto)
tick-18747f: in_progress → done (auto)
tick-fd039e: done (unchanged)
tick-c3e72b: done (unchanged)
tick-3d9a7e: done (unchanged)
```

The last three lines report nothing meaningful — those tasks were already done before the command ran. In projects with deeper hierarchies, unchanged lines can outnumber real transitions.

### Reproduction Steps

1. Create a parent task with multiple children
2. Complete some children individually
3. Complete the last remaining non-terminal child
4. Observe output includes "(unchanged)" entries for already-completed siblings

**Reproducibility:** Always

### Impact

- **Severity:** Low
- **Scope:** All users performing cascade transitions
- **Business impact:** Output clarity — no data correctness issue

### References

- Initial report identifies `buildCascadeResult` in `internal/cli/transition.go` as the source
- `CascadeResult` type in `internal/cli/format.go` carries `Unchanged []UnchangedEntry`
- All three formatter implementations render unchanged entries

---

## Analysis

### Initial Hypotheses

The `buildCascadeResult` function actively collects terminal descendants that weren't part of the cascade and populates them into the `Unchanged` slice. The formatters then dutifully render these entries. The fix likely involves either not collecting unchanged tasks, or not rendering them.

### Code Trace

*To be completed during analysis phase.*

### Root Cause

*To be completed during analysis phase.*

### Contributing Factors

*To be completed during analysis phase.*

### Why It Wasn't Caught

*To be completed during analysis phase.*

### Blast Radius

*To be completed during analysis phase.*

---

## Fix Direction

*To be completed after findings review.*

---

## Notes

*Additional observations during investigation.*
