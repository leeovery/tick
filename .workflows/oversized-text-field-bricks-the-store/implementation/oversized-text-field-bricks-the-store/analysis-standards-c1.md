AGENT: standards
FINDINGS: none
COMMENT_CORRECTIONS:
- internal/migrate/beads/beads.go:67 — the error contract is wrong: the body below also returns an error when issues.jsonl cannot be opened (`failed to open`) or read (`error reading`). That read-error return is in the loop this change rewrote.
  OLD: // Returns an error only if the .beads directory or issues.jsonl file is missing.
  NEW: // Returns an error if the .beads directory or issues.jsonl is missing or cannot be read.
SUMMARY: The production code matches every decision in the specification and its corrigenda:
- §2: the shared `jsonl.Lines` reader has no length ceiling, keeps the store's line rules byte for byte, and is used by both the store and doctor.
- §2.3: the beads reader has no length ceiling.
- §3: a line that fails to load is reported by line number, plus the task ID when the line has a string id.
- §4: rebuild parses before it touches the cache.
- §5: doctor runs every line through the store's own decoder. An incomplete read goes through `linesUnavailableResult`, and no rebuild advice is given while the store cannot load.
- §6: the 50,000-character description cap is enforced on create, update and migrate. Migrate also applies the CLI's title rules.

Verified by reading the code and by runs against a built binary: a bad timestamp line, a directory at the tasks.jsonl path, and the cap boundaries. The test suites, `go vet` and `golangci-lint` are all clean. No candidate cleared the finding floor. The one comment correction is a stale error-contract doc in the beads provider.
