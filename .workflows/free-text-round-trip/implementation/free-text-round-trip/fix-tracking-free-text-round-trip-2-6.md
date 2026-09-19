## Attempt 1

ISSUES:
- `internal/cli/readme_samples_test.go:188-199` — the `auto` column is not actually pinned where it matters. The simple-transition sample's `auto` value has no assertion at all (the subtest only reads `readmeCascadeAnchor`), so the README could document `auto,true` for a user-requested `tick start` — inverting the one distinction §7.2 exists to carry — with the suite green. And `got, _ := row["auto"].(bool)` makes the `want == false` branch vacuous: I confirmed against `toon-go` that a quoted `"false"` decodes to the string `"false"`, whose failed type assertion yields `false` and passes. The drift this task exists to prevent is exactly this, and the acceptance criterion names it ("`auto` reading `false`").
  FIX: drive the subtest from an expectation per anchor and compare the decoded value without a type assertion — `row["auto"] != want[i]` compiles for an `any` against a `bool` and fails on a string or a missing key:
  ```go
  for anchor, want := range map[string][]bool{
      readmeTransitionAnchor: {false},
      readmeCascadeAnchor:    {false, true},
  } {
      rows := readmeChangedRows(t, blocks, anchor)
      if len(rows) != len(want) {
          t.Fatalf("sample %q has %d rows, want %d", anchor, len(rows), len(want))
      }
      for i, row := range rows {
          if row["auto"] != want[i] {
              t.Errorf("sample %q row %d auto = %#v, want %v", anchor, i, row["auto"], want[i])
          }
      }
  }
  ```
  Rename the subtest to match its widened reach (e.g. "it marks only cascaded rows as automatic").
  ALTERNATIVE: leave the cascade subtest as-is and add a separate "it marks the requested transition as not automatic" subtest for `readmeTransitionAnchor` using the strict comparison. Smaller diff, but it leaves the cascade row-0 assertion vacuous against a type change, so I recommend the single widened subtest above.
  CONFIDENCE: high

NOTES:
- `README.md:465` — "The `auto` column marks the cascaded ones" holds for the TOON and JSON cells only; the Pretty cells have no `auto` column (they mark cascades with the `Cascaded:` tree). Accurate enough in context since both samples sit directly beneath, and not worth a fix round on its own, but a reader in pretty mode is told to look for a column that is not there.
- `internal/cli/readme_samples_test.go:224-232` — the JSON subtest finds its sample by "the one ```json fence whose body starts with `{`", which makes an unrelated future JSON object sample anywhere in the README fail this test with "README has 2 JSON object samples, want 1". It fails loudly rather than silently, so it is not a defect, but anchoring on the presence of a `"changed"` key would be less incidental.
- `.tick/tasks.jsonl` is modified in the working tree by the dogfooding status change only (the task row moving `open` → `in_progress`); no project data was touched.
