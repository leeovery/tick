# Cascade output shows unchanged tasks

When a status transition triggers cascades, the output includes lines for sibling and descendant tasks that were already in a terminal state and didn't actually change. These show up as "(unchanged)" entries alongside the real transitions, adding visual noise that makes it harder to see what actually happened.

For example, completing a task that has siblings already marked done produces output like:

```
$ tick done tick-b15fda
tick-b15fda: in_progress → done
tick-c5a1ff: in_progress → done (auto)
tick-18747f: in_progress → done (auto)
tick-fd039e: done (unchanged)
tick-c3e72b: done (unchanged)
tick-3d9a7e: done (unchanged)
```

The last three lines report nothing meaningful — those tasks were already done before the command ran. The expected behaviour is that only the primary transition and genuinely cascaded changes appear in the output. If a task didn't change, there's nothing to report.

The issue lives in `buildCascadeResult` in `internal/cli/transition.go`, which actively collects terminal descendants of involved tasks that weren't part of the cascade and populates them into the `Unchanged` slice of `CascadeResult`. The `CascadeResult` type in `internal/cli/format.go` carries an `Unchanged []UnchangedEntry` field, and all three formatter implementations — toon, pretty, and JSON — dutifully render these entries in their output.

This is low severity. The tool behaves correctly in terms of state management; it's purely an output clarity issue. But in projects with deeper hierarchies the unchanged lines can outnumber the real transitions, burying the useful information.
