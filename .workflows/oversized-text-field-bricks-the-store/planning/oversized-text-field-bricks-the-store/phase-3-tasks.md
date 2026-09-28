# Phase 3: Description cap — 4 tasks

## oversized-text-field-bricks-the-store-3-1

### Task 3-1: Description cap on `tick create`

**Problem**: A task description has no length limit. Titles are capped at 500 characters and notes at 2,000, but `tick create --description` accepts text of any size, and every later `tick show` returns it in full (§1.2, §6.1). Agents write the tasks, so nothing stops an agent from storing a description of any size. Once Phases 1 and 2 land, the store's safety no longer depends on field sizes (§2.1). The cap is hygiene, not a safety device: it keeps free text finite.

**Solution**: Cap the description at 50,000 characters, counted the way the title and note caps are: Unicode characters, not bytes, after leading and trailing whitespace is trimmed. The cap is a fixed constant alongside the title and note caps, not a flag. `tick create` checks it before anything is written. The refusal names the field, the limit, the length submitted and that nothing was saved, so the agent knows how much to cut and that it must retry the write.

**Outcome**: `tick create` accepts a description of up to exactly 50,000 characters. It refuses a longer one with exit 1, leaves the store untouched, and tells the agent everything it needs to cut the description and retry.

**Acceptance Criteria**:
- [ ] `tick create "T" --description` with exactly 50,000 characters exits 0, and `tick show` of the new task returns the full 50,000-character description (§6.1, §8.5)
- [ ] `tick create "T" --description` with exactly 50,000 multibyte characters, each several bytes in UTF-8, exits 0, and the full description is stored (§6.1, §8.5)
- [ ] `tick create "T" --description` with exactly 50,000 characters wrapped in leading and trailing whitespace (spaces, tabs, newlines) exits 0, and the description is stored trimmed (§6.1, §8.5)
- [ ] A store whose cache is current: `tick create "T" --description` with 50,001 characters exits 1. The error names the description field, the 50,000-character limit and the submitted length, 50,001, and says nothing was saved. `tasks.jsonl` and `cache.db` are byte-for-byte unchanged (§6.1, §6.2, §6.3, §8.5)
- [ ] `tick create "T" --description` with 50,001 multibyte characters wrapped in leading and trailing whitespace is refused, and the length it reports is 50,001: the characters after trimming, not the bytes and not the padded count (§6.1, §6.3)
- [ ] No flag is added for the cap. `tick help`, `tick help create`, the command flag registry and the README are unchanged (§6.1)
- [ ] Title and note refusals keep their current messages. `tick create` with a 501-character title still fails with `title exceeds maximum length of 500 characters`, and `tick note add` with 2,001 characters still fails with `note text exceeds maximum length of 2000 characters` (§7)

**Do**:
- The cap is a constant in `internal/task`, alongside the title cap (`maxTitleLen`, `internal/task/task.go:34`) and the note cap (`maxNoteTextLen`, `internal/task/notes.go:12`). It is counted as they are counted: characters of the trimmed text (§6.1).
- `RunCreate` (`internal/cli/create.go`) enforces it on `--description` before anything is written (§6.2). Today the description is only trimmed, with `task.TrimDescription` at `:218` inside the store mutation.
- Boundary tests follow the title boundary tests (`internal/task/task_test.go:76`, `:106`, `:122`) and their note counterparts in `internal/task/notes_test.go` (§8.5).

**Context**:
> §6.1: "A task description is capped at **50,000 characters**. It is counted the way the title (500) and note (2000) caps are: Unicode characters, not bytes, after leading and trailing whitespace is trimmed. A description of exactly 50,000 characters is accepted, multibyte ones included; 50,001 is refused. The cap is a fixed constant alongside the title and note caps, not a flag, so the command flag registry, `tick help` and the README do not change." The cap sits well above real use: the longest real description across the maintainer's tick stores is 20,785 characters. It is not sized against the 64 KiB line older binaries still enforce (§2.4).
>
> §6.3: "The refusal names the field, the limit, the length submitted, and that nothing was saved, so an agent knows how much to cut and that the write must be retried. For example: `description is 61,204 characters, over the 50,000-character limit; nothing was saved`. The length reported is the description's counted length, its characters after trimming (§6.1), so it compares directly with the limit. The four elements are required; the exact wording is the implementer's." Tasks 3-2 and 3-3 carry the same refusal on `update` and `migrate`.
>
> §7: nothing on the write path measures a record's encoded size, and the title and note refusals are not brought into line with §6.3. Title validation in `RunCreate` already runs before the store is opened (`internal/cli/create.go:123`).
>
> A description of 50,000 multibyte characters takes the task's line well past 65,536 bytes. It reads back because Phase 1 removed the reader's line-length ceiling (§2.1).
>
> Phase 1's tests that take a line past 65,536 bytes through `create --description` (Task 1-3) were planned to stay within 50,000 characters, using multibyte characters or JSON escape inflation. They keep passing under the cap.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.2, §1.4, §2.1, §2.4, §6.1, §6.2, §6.3, §7, §8.5

## oversized-text-field-bricks-the-store-3-2

### Task 3-2: Description cap on `tick update --description`

**Problem**: `tick update --description` is the second route that sets a description, and it has no limit either. With the cap on `create` alone, an agent could create a task with a short description and then set one of any size through `update` (§6.2). The cap must also leave alone tasks whose description is already over it. Such descriptions can come from before the cap, from a hand edit or from another tool, and those tasks must keep working (§6.2).

**Solution**: `tick update --description` enforces the same 50,000-character cap as `create` (Task 3-1), counted the same way, before anything is written, with the same refusal. The cap applies only to a description being set. A stored description already over it is left as it is, reads normally, and does not block other changes to its task.

**Outcome**: No `update` can set a description over 50,000 characters, and a refused update changes nothing. A task already holding an over-cap description can still be shown, retitled, noted and moved through its statuses.

**Acceptance Criteria**:
- [ ] An existing task: `tick update <id> --description` with exactly 50,000 characters exits 0 and stores the full description. The same holds for exactly 50,000 multibyte characters, and for exactly 50,000 characters wrapped in leading and trailing whitespace, which are stored trimmed (§6.1, §8.5)
- [ ] A store whose cache is current: `tick update <id> --description` with 50,001 characters exits 1. The error names the description field, the 50,000-character limit and the submitted length, 50,001, and says nothing was saved. `tasks.jsonl` and `cache.db` are byte-for-byte unchanged, and the task keeps its previous description (§6.2, §6.3, §8.5)
- [ ] `tick update <id> --title "New" --description` with 50,001 characters exits 1, and neither the title nor the description changes (§6.2, §6.3)
- [ ] `tick update <id> --description` with 50,001 characters wrapped in leading and trailing whitespace is refused, and the length it reports is 50,001 (§6.1, §6.3)
- [ ] A task whose stored description is 60,000 characters, written straight into `tasks.jsonl`: `tick show <id>` returns the full description, and `tick update <id> --title "New"` exits 0 with the title changed and the description unchanged (§6.2, §8.5)
- [ ] Same task: `tick start <id>` exits 0, and `tick note add <id> "text"` exits 0. After each, the description is unchanged (§6.2)

**Do**:
- `RunUpdate` (`internal/cli/update.go`) enforces the cap on `--description` before anything is written (§6.2). It validates `--description` today at `:194`, where it refuses an empty one. The cap and its counting are Task 3-1's constant in `internal/task`.

**Context**:
> §6.2: "Every route that sets a description enforces it … The check runs before anything is written, so a refused `create` or `update` exits 1 and leaves `tasks.jsonl` and the cache untouched." And: "The cap applies to a description being set. A description already stored over it is left as it is, reads normally (§2.1), and does not block other changes to its task — a status change, a new title, a note."
>
> §6.3's refusal is the same on `update` as on `create`: the field, the limit, the length submitted (the characters after trimming) and that nothing was saved, as the command's error with exit 1. The exact wording is the implementer's (Task 3-1).
>
> Phase 1's `update` tests past 65,536 bytes (Tasks 1-2 and 1-3) were planned so that no update sets a description over 50,000 characters. Task 1-2's fixture may hold an over-cap description written straight into `tasks.jsonl`, which this task leaves working.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §2.1, §6.1, §6.2, §6.3, §8.5

## oversized-text-field-bricks-the-store-3-3

### Task 3-3: Migrate skips an issue whose description exceeds the cap

**Problem**: `tick migrate` is the third route that sets a description, and it imports whatever the source holds. With the cap on `create` and `update` only, an import would still write a description of any size (§6.2). One oversized issue must not abort the whole import, and its description must not be cut short to fit (§6.4).

**Solution**: `tick migrate` checks each issue's description against the same 50,000-character cap (Task 3-1), counted the same way, before creating the issue. An issue over the cap is skipped and reported with its reason, exactly as the import already treats any other invalid issue. The reason carries the same four elements as the `create` and `update` refusal. The remaining issues import. `--dry-run` reports the same skip, because validation happens before the dry run's no-op creator is reached.

**Outcome**: No import writes a description over 50,000 characters. An over-cap issue is skipped with a reason an agent can act on, and it never stops the rest of the import.

**Acceptance Criteria**:
- [ ] A `.beads/issues.jsonl` holding, in order, an issue whose description is exactly 50,000 characters, one whose description is 50,001 characters and an ordinary issue: `tick migrate --from beads` completes the import as it does today when an issue is skipped, reporting `Done: 2 imported, 1 failed`. The first and third issues are in `tasks.jsonl`, the first with its full 50,000-character description (§6.4, §8.5)
- [ ] Same run: the second issue's skip line and its entry under `Failures:` give a reason naming the description field, the 50,000-character limit and the submitted length, 50,001, and saying nothing was saved. `tasks.jsonl` holds no task for it, neither whole nor truncated (§6.3, §6.4, §8.5)
- [ ] An issue whose description is exactly 50,000 characters wrapped in leading and trailing whitespace imports, with the description trimmed. An issue whose description is 50,001 characters wrapped likewise is skipped, and its reason reports 50,001 (§6.1, §6.3)
- [ ] `tick migrate --from beads --dry-run` on the same source reports the same skip with the same reason and the same summary, and writes nothing (§6.4, §8.5)

**Do**:
- An over-cap issue fails the validation the engine runs before creating each issue: `mt.Validate` at `internal/migrate/engine.go:74`, which is `MigratedTask.Validate` in `internal/migrate/migrate.go`. It is then skipped through the same branch as any other invalid issue and presented like any other skip (`internal/migrate/presenter.go`) (§6.4).
- The cap and its counting are Task 3-1's constant in `internal/task`.

**Context**:
> §6.4: "`tick migrate` validates each issue before creating it. An issue whose description exceeds the cap is skipped and reported with its reason, exactly as the import already treats any other invalid issue … The remaining issues import. The description is never truncated, and one oversized issue never aborts the import." And: "`--dry-run` reports the same refusals as a real run, because validation happens before the dry run's no-op creator is reached."
>
> §6.3: on `migrate` the refusal is the skipped issue's reason. It names the field, the limit, the length submitted (the characters after trimming) and that nothing was saved. The exact wording is the implementer's.
>
> The engine trims each issue's description (`mt.Normalize()`, using `task.TrimDescription`) before it validates. Today a skipped issue gets a `✗ Task: <title> (skipped: <reason>)` line, is counted as failed in the `Done: N imported, M failed` summary and is listed under `Failures:`. `tick migrate` still exits 0 when issues are skipped (see `internal/cli/migrate_test.go`).
>
> Phase 1's long-line beads test (Task 1-5) was planned to keep its long issue within the description and title limits, so it keeps importing under this task.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §6.1, §6.2, §6.3, §6.4, §8.5

## oversized-text-field-bricks-the-store-3-4

### Task 3-4: Migrate adopts the CLI's title rules

**Problem**: `tick migrate` checks only that an imported title is non-empty (`MigratedTask.Validate`, `internal/migrate/migrate.go`). The CLI refuses a title over 500 characters or spanning more than one line. So an import can write a title that no `tick create` or `tick update` would accept (§6.4).

**Solution**: Migrate adopts the CLI's title rules. An issue whose title exceeds 500 characters or spans more than one line is skipped with its reason, the same way as any other invalid issue. The reason is the CLI's current title refusal message. The remaining issues import, and `--dry-run` reports the same skips.

**Outcome**: Every title migrate writes is one the CLI would accept. An issue whose title the CLI would refuse is skipped with the CLI's reason, and the rest of the import proceeds.

**Acceptance Criteria**:
- [ ] A `.beads/issues.jsonl` holding an issue with a 501-character title, an issue whose title spans two lines, an issue with a title of exactly 500 characters and an ordinary issue: `tick migrate --from beads` skips the first two and imports the other two, reporting `Done: 2 imported, 2 failed` (§6.4, §8.5)
- [ ] Same run: the 501-character title's reason is `title exceeds maximum length of 500 characters`, and the two-line title's reason is `title must be a single line (no newlines)`. These are the CLI's current title refusal messages (§6.4, §7)
- [ ] An issue whose title is exactly 500 multibyte characters imports, because the title is counted in characters, not bytes (§1.2, §6.4)
- [ ] An issue whose title has line breaks only at its edges, such as `"Ship it\n"`, imports with the title `Ship it`, as the CLI accepts it once trimmed (§6.4)
- [ ] `tick migrate --from beads --dry-run` on the same source reports the same two skips with the same reasons (§6.4)

**Do**:
- `MigratedTask.Validate` (`internal/migrate/migrate.go`) checks only for an empty title today. The engine calls it before creating each issue (`internal/migrate/engine.go:74`), so an issue that fails it is skipped through the existing branch (§6.4).
- The CLI's title rules and refusal messages are `task.ValidateTitle` (`internal/task/task.go:177`) (§6.4, §7).

**Context**:
> §6.4: "Migrate also adopts the CLI's title rules. Today it checks only that the title is non-empty. An issue whose title exceeds 500 characters or spans more than one line is now skipped the same way, with its reason." And: "`--dry-run` reports the same refusals as a real run, because validation happens before the dry run's no-op creator is reached."
>
> §7: "Title and note refusals keep their current messages." The CLI's messages are `title is required and cannot be empty`, `title must be a single line (no newlines)` and `title exceeds maximum length of 500 characters`. Migrate's empty-title message is already the CLI's.
>
> The engine trims each title (`mt.Normalize()`, using `task.TrimTitle`) before it validates, as the CLI trims before validating. The existing engine tests `it reports the trimmed title in a dry run` and `it keeps today's error for a whitespace-only title` (`internal/migrate/engine_test.go`) keep holding. Today an issue with a long or multi-line title imports with that title stored as it is.
>
> Phase 1's long-line beads test (Task 1-5) was planned to keep its long issue within the title and description limits.

**Spec Reference**: `.workflows/oversized-text-field-bricks-the-store/specification/oversized-text-field-bricks-the-store/specification.md` — §1.2, §6.4, §7, §8.5
