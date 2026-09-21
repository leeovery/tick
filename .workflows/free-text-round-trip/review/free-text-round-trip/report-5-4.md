TASK: free-text-round-trip-5-4 (tick-b70d39) — Tick Migrate Trims Imported Free Text

ACCEPTANCE CRITERIA:
- An imported title carrying leading or trailing whitespace is stored trimmed
- An imported description carrying leading or trailing whitespace, including a leading newline, is stored trimmed
- A whitespace-only description imports as the empty string and the import succeeds
- No import fails because a description was whitespace-only
- A whitespace-only title still fails with today's `title is required and cannot be empty` and is reported under the `(untitled)` fallback title
- Interior whitespace, blank lines and newlines inside a description are preserved byte-for-byte
- A dry run reports the trimmed titles, because it shares the Engine
- Normalisation happens at exactly one point, so a provider that later carries note text inherits the rule without a second decision
- The providers under `internal/migrate/beads/` are unchanged
- No pass rewrites stored records: a task already in `tasks.jsonl` carrying edge whitespace is byte-identical after a migration runs
- `tick show` emits stored bytes unmodified — a stored description carrying edge whitespace comes back with that whitespace
- `go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean

STATUS: complete

SPEC CONTEXT: §2.2 states the fidelity bar as byte-identity and names the invariant it rests on — every path that stores free text trims edge whitespace before storing. `create`/`update`/`note add` already do; `tick migrate` was the hole, storing the source tool's value as it arrives (`store_creator.go` building `Title: mt.Title` / `Description: mt.Description` while `Validate` trimmed only for its emptiness check). The spec requires migrate to trim on the way in, as a normalisation and not a validation (whitespace-only stores empty, no import fails for it), and draws a hard boundary: values already in storage are left alone and `tick show` never emits anything but the stored bytes. §3.3 confirms only migrate's *write* path is touched here; its printed output is unchanged.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/migrate/migrate.go:56-62 — `MigratedTask.Normalize()` returns a value copy with `task.TrimTitle(mt.Title)` and `task.TrimDescription(mt.Description)`, every other field untouched (value receiver, so the caller's copy is not mutated).
  - internal/migrate/engine.go:71-72 — loop variable renamed off `task` to `mt` (the package shadow the plan called out is gone) and reassigned from `Normalize()` before anything else in the body.
  - internal/migrate/engine.go:74-85 — `Validate()`, `CreateTask()` and all three `Result.Title` values now read the normalised `mt`; the empty-title fallback collapsed to `cmp.Or(mt.Title, FallbackTitle)`, which is exact after normalisation since a whitespace-only title normalises to `""`.
- Notes:
  - Single-point claim verified by enumeration, not assumption: `CreateTask(` has exactly two implementations (internal/migrate/store_creator.go:33, internal/migrate/dry_run_creator.go:12), one interface declaration (internal/migrate/engine.go:38) and exactly one non-test call site (internal/migrate/engine.go:80). `Validate()` has exactly one non-test call site (internal/migrate/engine.go:74), and `Normalize()` exactly one (internal/migrate/engine.go:72). Both creators therefore inherit the rule, which is what makes the dry-run path report the same titles the real run stores — internal/cli/migrate.go:106-121 selects between them and hands both to the same `Engine`.
  - Scope boundary confirmed: `grep -rn 'TrimTitle|TrimDescription|TrimSpace' internal/migrate/ --include='*.go'` minus tests returns migrate.go:44 (Validate's pre-existing emptiness check), migrate.go:59-60 (Normalize) and beads/beads.go:87 (line-level trim while scanning JSONL, unrelated to free-text storage). `git log -- internal/migrate/beads/` shows its last touch is a0bf228a, predating this work — the providers are unchanged.
  - No rewrite pass: nothing outside `CreateTask` builds a `task.Task` in the migrate path, so stored records are only re-serialised by `Store.Mutate`, from the same `MarshalJSONL` that wrote them.
  - Read path untouched: `tick show --field description` returns `d.Task.Description` bare (internal/cli/show_fields.go:60) and prints it through `fmt.Fprintln` (internal/cli/show.go:73) — no trim on the way out, as §2.2 requires.

TESTS:
- Status: Adequate
- Coverage:
  - internal/migrate/engine_test.go:900-1077 (`TestEngineNormalization`) covers title trim (asserting both the value reaching the creator and the reported `Result.Title`), description trim, whitespace-only description importing as empty and succeeding, the imported/failed counts, the preserved whitespace-only-title error and `(untitled)` fallback, interior whitespace and blank lines preserved byte-for-byte, the dry-run reported title via `DryRunTaskCreator`, and normalise-before-validate (asserting zero `CreateTask` calls).
  - internal/migrate/engine_test.go:1079-1110 (`TestMigratedTaskNormalize`) pins the field-level contract: every non-free-text field survives and the receiver is not mutated.
  - internal/cli/migrate_test.go:673-694 drives the real CLI over a beads fixture and asserts the *persisted* title and description through `readPersistedTasks` — this is the one that would catch a regression where the Engine normalised but `StoreTaskCreator` re-read an untrimmed source.
  - internal/cli/migrate_test.go:696-721 seeds a task carrying edge whitespace via `setupTickProjectWithTasks`, captures its raw `tasks.jsonl` line before the migration and compares after — the boundary criterion, asserted at the byte level on the line rather than inferred.
  - internal/cli/migrate_test.go:723-750 runs `tick show <id> --field description` on that seeded record and asserts stdout equals the stored bytes plus the `Fprintln` newline — a trim added on the read path would fail here.
  - Tests would fail if the feature broke: remove the `mt = mt.Normalize()` line and the creator-argument assertions, the `Result.Title` assertions and both end-to-end assertions all fail.
- Notes:
  - Mild overlap between "it imports a whitespace-only description as empty" and "it does not fail an import because the description was whitespace-only" — the first already asserts `results[0].Success`. Both subtests are named individually in the plan's Tests list, so this is compliance with the plan rather than drift, and the second is the only one framed over the imported/failed counts the presenter reports. Not worth collapsing.
  - `TestMigratedTaskNormalize` compares whole structs with `!=`, including `time.Time` fields. Safe here: both sides carry the same `time.Date` value with the same location pointer and no monotonic reading, so the comparison is deterministic.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing`, `t.Run()` subtests named "it does X", `t.Helper()` on the new `readJSONLLine` helper, `t.TempDir()` isolation via the existing setup helpers. `Normalize` matches the codebase's American spelling used elsewhere (`task.NormalizeID`).
- SOLID principles: Good — the rule lives on the type that owns the data and is applied once at the orchestration point, so both `TaskCreator` implementations inherit it rather than each deciding.
- Complexity: Low — one added method, one added statement in `Run`, and a four-line conditional replaced by `cmp.Or`.
- Modern idioms: Yes — `cmp.Or` for the fallback, `strings.SplitSeq` in the new test helper, value-receiver copy-and-return rather than a pointer mutator.
- Readability: Good — `mt` removes the `task`-package shadow the plan flagged, and the normalise statement sits alone at the top of the loop body where its ordering relative to `Validate` is visible.
- Issues: None. The doc comments touched by this work hold against the code: `Run`'s comment (internal/migrate/engine.go:54-59) now states the trim, and the stale "normalized" claims on `MigratedTask` and `Provider.Tasks` were corrected in 53a007b2 to match where normalisation actually happens.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...` and `gofmt -l ./internal ./cmd` are clean" — settling this needs those three commands run over the working tree; reading cannot confirm a clean vet or an empty gofmt list.
