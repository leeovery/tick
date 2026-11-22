# Rename Misleading Dep Tree Test Name

The test subtest "it outputs no dependencies found for project with no tasks" in `internal/cli/dep_tree_test.go` is misleading. The test body actually creates tasks that have no dependency relationships between them — it's not testing an empty project at all.

The current name implies the project has zero tasks, when the intent is to verify that a project with tasks but no dependency edges outputs the "No dependencies found." message. Something like "it outputs no dependencies found for project with tasks but no deps" would more accurately describe what's being tested.

This was surfaced during the dep-tree-visualization review. It's a cosmetic naming issue with no functional impact, but test names are documentation — when a test fails, the name is the first thing you read to understand what broke. A misleading name slows down diagnosis.
