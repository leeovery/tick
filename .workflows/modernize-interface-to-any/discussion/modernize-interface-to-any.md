# Discussion: Modernize interface{} to any

## Context

Go 1.18 introduced `any` as a built-in alias for `interface{}`. The Tick codebase still uses `interface{}` in several places -- notably in `internal/cli/json_formatter.go` where `marshalIndentJSON` accepts `interface{}` as its parameter type. Meanwhile, newer generic code already uses `any` (e.g., `writeTree[T any]`), creating an internal inconsistency.

This is a purely mechanical, cosmetic change -- the compiler treats `interface{}` and `any` identically. The goal is to align the codebase with modern Go idioms and eliminate the inconsistency between older and newer code.

### References

- [Go 1.18 release notes](https://go.dev/doc/go1.18) -- introduced `any` as alias for `interface{}`
- Inbox item: `.workflows/.inbox/.archived/ideas/2026-03-28--modernize-interface-to-any.md`

## Questions

- [ ] What is the full scope of `interface{}` usage across the codebase?
- [ ] Should we do a blanket find-and-replace, or are there cases where `interface{}` should be preserved?
- [ ] Should this include comments and documentation, or just code?

---

*Each question above gets its own section below. Check off as completed.*

---

## What is the full scope of interface{} usage across the codebase?

### Context
Need to understand how widespread `interface{}` is before deciding on approach.

---

## Should we do a blanket find-and-replace, or are there cases where interface{} should be preserved?

### Context
While `any` and `interface{}` are semantically identical, there could be edge cases worth considering (e.g., generated code, third-party interfaces).

---

## Should this include comments and documentation, or just code?

### Context
Comments or doc strings may reference `interface{}` as a Go concept. Need to decide whether to update those too.
