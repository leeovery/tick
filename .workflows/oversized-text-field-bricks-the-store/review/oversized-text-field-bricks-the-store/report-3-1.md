TASK: Description Cap On Tick Create (oversized-text-field-bricks-the-store-3-1, tick-70bb02)

ACCEPTANCE CRITERIA:
- `tick create "T" --description` with exactly 50,000 characters exits 0, and `tick show` of the new task returns the full 50,000-character description (§6.1, §8.5)
- `tick create "T" --description` with exactly 50,000 multibyte characters, each several bytes in UTF-8, exits 0, and the full description is stored (§6.1, §8.5)
- `tick create "T" --description` with exactly 50,000 characters wrapped in leading and trailing whitespace (spaces, tabs, newlines) exits 0, and the description is stored trimmed (§6.1, §8.5)
- A store whose cache is current: `tick create "T" --description` with 50,001 characters exits 1. The error names the description field, the 50,000-character limit and the submitted length, 50,001, and says nothing was saved. `tasks.jsonl` and `cache.db` are byte-for-byte unchanged (§6.1, §6.2, §6.3, §8.5)
- `tick create "T" --description` with 50,001 multibyte characters wrapped in leading and trailing whitespace is refused, and the length it reports is 50,001: the characters after trimming, not the bytes and not the padded count (§6.1, §6.3)
- No flag is added for the cap. `tick help`, `tick help create`, the command flag registry and the README are unchanged (§6.1)
- Title and note refusals keep their current messages. `tick create` with a 501-character title still fails with `title exceeds maximum length of 500 characters`, and `tick note add` with 2,001 characters still fails with `note text exceeds maximum length of 2000 characters` (§7)

STATUS: complete

SPEC CONTEXT: §6.1 caps a description at 50,000 Unicode characters counted after trimming, as a fixed constant beside the title (500) and note (2000) caps, with no flag, help or README change. §6.2 requires the check before anything is written, so a refused create exits 1 with tasks.jsonl and the cache untouched. §6.3 requires the refusal to carry four elements (field, limit, submitted trimmed length, nothing saved), wording left to the implementer. §7 freezes the title and note refusal messages. §8.5 names the boundary tests, with the title tests in internal/task/task_test.go as the pattern.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/task/task.go:35 — `maxDescriptionLen = 50000` in the same const block as `maxTitleLen` (:34)
  - internal/task/task.go:202-210 — `ValidateDescription` counts `utf8.RuneCountInString(strings.TrimSpace(desc))` and refuses with `description is %d characters, over the %d-character limit; nothing was saved`
  - internal/cli/create.go:128-130 — `RunCreate` calls `task.ValidateDescription(opts.description)` right after title validation and before `openStore` (:162), so a refusal touches neither tasks.jsonl nor cache.db
  - internal/cli/create.go:222 — stored value still goes through `task.TrimDescription`, which applies the same `strings.TrimSpace` as the count, so the counted and stored text agree
- Notes: All four §6.3 elements are present. The message prints the numbers without thousands separators (`50001`, `50000-character`), where the spec's example has them. §6.3 leaves the wording to the implementer, so this is not drift. No flag was added: help.go, flags.go and README.md are not in the change-set. The title and note validators are untouched (ValidateTitle at task.go:178-190 keeps its message). The Phase 1 reader (internal/jsonl/lines.go:29-48, bufio.Reader.ReadBytes) has no line ceiling, so the ~200 KB multibyte line reads back.

TESTS:
- Status: Adequate
- Coverage:
  - Unit, internal/task/task_test.go:674-722 (TestValidateDescription): 50,000 accepted; 50,001 refused with the exact message asserted; 50,000/50,001 four-byte characters (U+1F600); whitespace-padded 50,000 accepted; padded 50,001 three-byte characters reporting the trimmed count 50001; empty accepted. These follow the title boundary pattern §8.5 names.
  - CLI, internal/cli/create_test.go:1505-1628 (TestCreateDescriptionCap):
    - Table-driven accepted cases (:1516-1555) cover exactly 50,000 ASCII, 50,000 four-byte multibyte, and 50,000 wrapped in space/tab/newline. Each asserts exit 0, the persisted description and the `tick show` (toon) description in full, with the padded case stored trimmed (criteria 1-3).
    - :1557-1584 seeds through a real create, so cache.db exists and is current. It then asserts exit 1, the exact stderr carrying all four elements, empty stdout, and byte-for-byte equality of tasks.jsonl and cache.db (criterion 4).
    - :1586-1600 sends padded 50,001 three-byte characters and asserts the reported count is 50001 and nothing was persisted (criterion 5).
    - :1602-1627 pins the unchanged title and note refusal messages exactly (criterion 7).
  - Every test fails if the cap is removed, if the limit shifts by one, or if bytes or the padded length are counted instead of trimmed characters.
- Notes: The unit and CLI tests overlap by layer, not by redundancy. The plan asks for internal/task boundary tests, and the criteria are CLI outcomes. Criterion 6 is a negative ("unchanged"). Nothing on the help, flags or README side was touched, and the existing flag-registry drift test and README sample tests guard it.

CODE QUALITY:
- Project conventions: Followed (validator in internal/task beside its siblings, stdlib testing with t.Run subtests and "it ..." naming, t.Helper on helpers, toon output decoded rather than string-matched)
- SOLID principles: Good
- Complexity: Low
- Modern idioms: Yes
- Readability: Good
- Issues: None

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- None
