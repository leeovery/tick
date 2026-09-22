# Specification: Same-Second Tasks Sort By ID

## Specification

### 1. The Ordering Contract

#### 1.1 The defect

Tasks created inside one wall-clock second come back in an order unrelated to the order they were written. There is no error and no warning — the ordering is silently wrong. A human typing `tick add` repeatedly never reaches it; anything scripted reaches it every time: an agent authoring a whole phase in one pass, the migration framework importing a project, a shell loop. Machine writers are now the normal case.

Two independent defects compound. Neither alone produces the symptom; together they mean authoring order is sometimes preserved and sometimes scrambled, with nothing to tell the two cases apart.

**The ordering information is never recorded.** Creation times are stored to whole-second granularity — the single format constant governing both storage and display names no fractional part (`sed -n 41p internal/task/task.go` → `const TimestampFormat = "2006-01-02T15:04:05Z"`), and every stamping site additionally truncates before the value is held (`rg -c 'Truncate\(time\.Second\)' internal/ --glob '!*_test.go'` → 9 sites). A batch authored inside one second records the same instant for every member. The intended order is not degraded in storage — it is absent from it.

**The sort has no deterministic final term.** Both list-family clauses end at `t.created ASC` (`rg -n 'ORDER BY' internal/cli/list.go` → `:317` ready view, `:319` neutral view). SQLite is free to emit tied rows in any order, and emits them in whatever order the chosen plan feeds the sorter. Three plans were observed across measured projects: the priority index and the status index both feed rows in rowid order, which is file order — the right answer, reached by accident; the primary-key index feeds rows in task-ID order — the reported symptom, and since task IDs are three random bytes, indistinguishable from a shuffle.

**Which plan a query gets is not predictable.** It turns on data shape and statistics, not on command name or row count: `tick ready --parent` returned file order in one 8-task project and ID order in 7-task and 18-task projects on the same schema and binary, and `ANALYZE` changed nothing. The correct mental model is not "parent-filtered reads are broken" but **tie order is undefined everywhere, and currently resolves correctly by chance in most cases**. A new index, a statistics change or a SQLite version bump can flip any of it silently.

#### 1.2 Severity

`--count` turns wrong order into wrong *selection*. `LIMIT` is appended after the tied `ORDER BY` (`internal/cli/list.go:321-324`), so it cuts the arbitrary order rather than the authored one. Measured: in a five-child phase authored step-1 … step-5, `tick ready --parent P --count 1` returned **step-4**. An implementation loop asking for the next available task is handed the wrong task outright, not a list it could re-sort.

No live exposure — no stored batch is currently being read in the wrong order, and nothing is being worked around by hand.

#### 1.3 What tick guarantees after the fix

Tasks come back in the order they were created, within a priority band — and within the `in_progress` band for `ready`, which floats above the priority terms. A batch written one after another by an agent, an importer or a shell loop reads back in the sequence it was written.

The guarantee is **total**: every query in the list family produces one defined order under every condition, with no dependence on which plan SQLite chooses. A mixed-priority batch still does not read back in write order — priority and the `ready` band are semantic ordering and continue to outrank creation order.

Creation time stops being load-bearing for ordering. The timestamp format is unchanged, and no stored timestamp changes value.

---

## Working Notes
