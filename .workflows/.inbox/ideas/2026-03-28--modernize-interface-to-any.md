# Modernize interface{} to any Across the Codebase

Go 1.18 introduced `any` as a built-in alias for `interface{}`. The Tick codebase still uses `interface{}` in several places — notably in `internal/cli/json_formatter.go` where `marshalIndentJSON` accepts `interface{}` as its parameter type. This is a pre-existing pattern that predates the dep-tree-visualization feature and likely appears in other files as well.

Replacing `interface{}` with `any` is a straightforward mechanical change that aligns the codebase with modern Go idioms. It's purely cosmetic — the compiler treats them identically — but it improves readability and signals that the project follows current conventions. The newer generic code in the same codebase already uses `any` (e.g., `writeTree[T any]`), so there's an internal inconsistency worth cleaning up.

This would be a good candidate for a cross-cutting cleanup pass: find all `interface{}` usages, replace with `any`, verify tests pass.
