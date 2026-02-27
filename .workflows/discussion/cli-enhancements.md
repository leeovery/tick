---
topic: cli-enhancements
status: in-progress
work_type: feature
date: 2026-02-27
---

# Discussion: CLI Enhancements

## Context

Six feature additions bundled as one feature, all from the IDEAS.md planned list. These are additive enhancements to the existing CLI with no architectural impact — new fields on Task, new flags on commands, and one new subcommand.

1. **List Count/Limit** — `--count N` flag on `list`/`ready`/`blocked` to cap results (LIMIT clause)
2. **Partial ID Matching** — prefix matching on hex portion of task IDs, error on ambiguous match
3. **Task Types** — `bug`, `feature`, `task`, `chore` string field with validation + filtering
4. **External References** — `[]string` field for cross-system links (`gh-123`, `JIRA-456`, URLs)
5. **Tags** — `[]string` field with `--tags` on create/update, `--tag` filter on list
6. **Notes** — timestamped text entries appended to a task (subcommand)

### References

- [IDEAS.md](../../IDEAS.md) — source of all six items

## Questions

- [ ] How should new fields (tags, type, refs) be stored in JSONL and cached in SQLite?
- [ ] What's the right UX for partial ID matching — where does resolution happen?
- [ ] How should Notes work as a subcommand — add/list/show?
- [ ] Should tags and type be settable at creation only, or also via update?
- [ ] How should filtering work for tags and type on list commands?
- [ ] What validation rules apply to task types and tags?

---

*Each question above gets its own section below. Check off as concluded.*

---
