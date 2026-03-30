AGENT: standards
FINDINGS: none
SUMMARY: Implementation conforms to specification and project conventions. The qualifyCommand fix correctly guards "tree" to dep-only (app.go:377-381), all five acceptance criteria are covered by tests across dep_tree_test.go and note_test.go, and all three spec-required unit tests for qualifyCommand are present. Shared subcommands (add, remove) remain unaffected. No project skill MUST DO/MUST NOT DO violations found.
