TASK: free-text-round-trip-6-5 (tick-2974b8) — JSON Output Is Parsed And Asserted By Decoded Value

ACCEPTANCE CRITERIA:
- Every non-exempt entry in `conformanceDocs` is run under `--json` and parses as exactly one JSON value, with no second value in the stream
- The decoded value is an object for every document except `list`, `ready` and `blocked`, whose documents are top-level arrays
- `changed`, `roots`, `blocked_by`, `blocks`, `tags`, `refs`, `notes`, `children` and `by_priority` unmarshal to non-nil slices wherever they appear, never `null`
- Every `auto` value unmarshals as a Go `bool` and every `index` value as a number
- A bare task's JSON detail carries `type`, `tags`, `refs` and `description` and carries no `parent` or `closed`, unlike the same task's toon document
- A filtered JSON document carries exactly the selected keys
- `stats` JSON keeps its `{total, by_status, workflow, by_priority}` shape
- Both drivers iterate the same `conformanceDocs`, so adding an entry adds it to both formats
- Pretty is in neither driver
- `go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean

STATUS: complete

SPEC CONTEXT:
§11 takes the no-byte-level-pinning trade "for toon and JSON" and requires rewritten assertions to check decoded values rather than output text; pretty is explicitly excluded because it has no parser. §4.2 requires a JSON consumer to get the same structured answer as a TOON one (the `changed` list, the structured empty dep-tree form, the 1-based note index). §7.4 is the load-bearing property this task's driver checks document by document: "the stream must be one document" — making each section valid is not sufficient. §3.1 sets the must-parse inventory and its two exemptions (a bare `--field` value, and a selection printing no bytes), and §3.2 records that under `--json` the five prose commands return objects, so §11's must-parse coverage reaches them there — which is why the JSON driver runs a superset of the toon driver's entries.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - `internal/cli/conformance_test.go:1692-1696` — `TestJSONOutputConformance` drives the whole inventory under the JSON driver
  - `internal/cli/conformance_test.go:711-726` — `decodeSingleJSONValue`: first `Decode` into `any` must succeed, the next must return `io.EOF`; a second value and trailing garbage each produce a distinct error naming the entry and quoting the stream
  - `internal/cli/conformance_test.go:731-748` — `jsonTopLevelIsArray` / `jsonShapeProblem`: `[]any` for `list`, `ready`, `blocked`; `map[string]any` for everything else
  - `internal/cli/conformance_test.go:702-706, 753-796` — `jsonConformanceListKeys` plus the `jsonInvariantProblems` / `jsonMemberProblem` / `assertJSONInvariants` walk over every nested value
  - `internal/cli/conformance_test.go:800-824` — `runJSONConformanceDoc` (run, parse-one, shape, invariants) and `decodeJSONConformanceEntry`
  - `internal/cli/conformance_test.go:654-692` — `conformanceSkipReason` / `driveConformanceInventory` / `drivenConformanceEntries`: one inventory, two drivers, exemptions declared per format
  - `internal/cli/conformance_test.go:1820-1896` — the per-document JSON subtests (task lists as arrays, detail presence rules, filtered key set, nested stats shape)
- Notes:
  - The criterion's `roots` key is now `trees` in the invariant list (`internal/cli/conformance_test.go:703`). That is a tracked, sound divergence: task 7-2 (commit `777f67e3`) renamed the full-graph JSON key, and `jsonDepTreeFull` declares `json:"trees"` at `internal/cli/json_formatter.go:337`. The list also gained `removed` and `deps_updated` (task 6-8) when the prose commands joined the JSON driver. I enumerated every list-valued key the JSON formatter emits — `changed`, `trees`, `children`, `blocked_by`, `blocks`, `tags`, `refs`, `notes`, `by_priority`, `removed`, `deps_updated` — and all eleven are in `jsonConformanceListKeys`; no key in that list is dead.
  - The planned test `"it drives both formats from one inventory"` became `"it skips a toon-exempt entry in the toon driver and runs it under json"` (`internal/cli/conformance_test.go:1793-1817`) when task 6-8 made the JSON driver a superset of the toon driver. It still asserts both drivers walk `conformanceDocs`, so the criterion holds in substance.
  - Shape gating keys off `entry.Command`, which task 6-7 pinned to the arguments the setup actually runs (`declaredCommandProblem`, `internal/cli/conformance_test.go:829-837`), so the array/object exemption cannot be claimed by an entry running a different command.
  - The one-value-per-stream property matches production: every handler prints its document exactly once (`internal/cli/helpers.go:36-42`, `internal/cli/show.go:88-90`, `internal/cli/dep.go:125,195`, `internal/cli/remove.go:194`, `internal/cli/init.go:37`, `internal/cli/rebuild.go:26`).
  - Scope respected: the task's commit `6d8000da` touched `internal/cli/conformance_test.go` only; `internal/cli/json_formatter_test.go` was not modified.

TESTS:
- Status: Adequate
- Coverage:
  - Driver-level: every inventory entry carrying a `Setup` is run under `--json`, parsed as one value, shape-checked, and walked for invariants (`internal/cli/conformance_test.go:1692-1696, 800-812`).
  - Check-level fixtures exercise each rule in isolation without a CLI run: one value then EOF, two concatenated objects rejected (`:1699-1724`), array vs object per command (`:1726-1752`), a document meeting every invariant including nested `children` (`:1754-1765`), a null under each of the eleven list keys (`:1767-1775`), a string `auto` (`:1777-1783`), a string `index` (`:1785-1791`).
  - Document-level: task lists decode to non-empty `[]any` (`:1820-1841`); a bare task's detail carries `type`, `tags`, `refs`, `description` and omits `parent`/`closed` (`:1864-1880`); a filtered document's key set equals the selection (`:1882-1886`); stats keeps `{total, by_status, workflow, by_priority}` (`:1889-1896`).
  - Would fail if the feature broke: dropping a section, emitting `null` for a list, quoting `auto`/`index`, or appending a second document to a stream each fail a named assertion. The invariant rules are unit-tested against fixtures, so they are not vacuous.
- Notes:
  - `TestJSONTaskListConformance` re-runs three entries the driver already covers, but it adds an assertion the driver does not make (the decoded array is non-empty), so it is not redundant.
  - The invariant keys are all exercised by real runs: `tags`/`refs`/`notes`/`children`/`blocked_by` via the `show` entries, `changed` via the status and mutation entries, `trees`/`blocks` via the `dep tree` entries, `by_priority` via `stats`, `removed`/`deps_updated` via `remove`.

CODE QUALITY:
- Project conventions: Followed — stdlib `testing` only, `t.Run` subtests named "it does X", `t.Helper()` on every helper, `t.TempDir()`-backed setups, no assertion library.
- SOLID principles: Good — `decodeSingleJSONValue`, `jsonShapeProblem` and `jsonInvariantProblems` are pure string/value functions returning problems, with the `*testing.T` coupling confined to the thin `assertJSONInvariants` and `runJSONConformanceDoc` wrappers; that is what makes the checks themselves unit-testable.
- Complexity: Low — the invariant walk is one type switch plus one per-key switch; the driver split (`driveConformanceEntry` taking a `drive` func) removed the duplication that a second inventory loop would have created.
- Modern idioms: Yes — `slices.Sorted(maps.Keys(...))` for deterministic problem order, `slices.Contains`, `errors.Is(err, io.EOF)`, `switch` on the decode result rather than nested ifs.
- Readability: Good — failures name the entry and quote the stream, so a driver failure is diagnosable from the test log alone.
- Issues: None. The JSON tests reuse the `assertToon*`/`toonRows` helpers from `internal/cli/toon_decode_test.go:37-88`; those helpers operate on `map[string]any` and are format-agnostic in fact, and renaming them is a preference, not a defect.

BLOCKING ISSUES:
- None

FINDINGS:
- None

UNSETTLED:
- "`go test ./...`, `go vet ./...`, `gofmt -l ./internal ./cmd` and `golangci-lint run ./...` are clean" — settled only by running those four commands; reading cannot establish that the suite passes or that the linters are silent. In particular, whether every inventory entry's real `--json` stdout parses as exactly one value and satisfies the invariants is asserted at runtime by `TestJSONOutputConformance`, and only an executing pass measures it.
