TASK: Update CLI callers to use StateMachine methods

ACCEPTANCE CRITERIA:
- All existing callers in internal/cli/ updated to use StateMachine methods
- All existing tests pass with no regressions

STATUS: Complete

SPEC CONTEXT: The specification defines a StateMachine struct that consolidates all 11 cascade/validation rules. Existing logic from transition.go and dependency.go migrates into StateMachine. Old functions become thin wrappers or get deleted. Callers update accordingly.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - internal/cli/transition.go:32 — `var sm task.StateMachine` with `sm.ApplyWithCascades()`
  - internal/cli/dep.go:113 — `var sm task.StateMachine` with `sm.ValidateAddDep()`
  - internal/cli/create.go:223 — `var sm task.StateMachine` with `sm.ValidateAddDep()` and `validateAndReopenParent()` (which uses `sm.ValidateAddChild()` and `sm.ApplyWithCascades()`)
  - internal/cli/update.go:292 — `var sm task.StateMachine` with `sm.ValidateAddDep()`, `validateAndReopenParent()`, and `evaluateRule3()` (which uses `sm.ApplyWithCascades()`)
  - internal/cli/helpers.go:114 — `validateAndReopenParent()` helper uses `sm.ValidateAddChild()` and `sm.ApplyWithCascades()`
- Notes: Zero remaining direct calls to `task.Transition()` or `task.ValidateDependency()` in internal/cli/. Old functions in task package are thin wrappers delegating to StateMachine, preserving backward compatibility for any external callers or domain-level tests.

TESTS:
- Status: Adequate
- Coverage: CLI tests (transition, dep, create, update) exercise StateMachine indirectly through the command handlers. The helpers_test.go file has direct StateMachine tests for validateAndReopenParent at lines 417-501. No separate "migration test" needed since the acceptance criteria is about caller migration, not new behavior.
- Notes: Existing tests cover the same behavior paths through the new StateMachine entry points. If StateMachine methods broke, CLI tests would fail.

CODE QUALITY:
- Project conventions: Followed — stdlib testing, t.Run subtests, error wrapping with %w, functional Store.Mutate pattern
- SOLID principles: Good — StateMachine is a clean abstraction point; CLI callers depend on its interface rather than scattered functions
- Complexity: Low — each caller creates `var sm task.StateMachine` and calls the appropriate method; no complex indirection
- Modern idioms: Yes — value-type StateMachine (no constructor needed), consistent with Go's stateless method receiver pattern
- Readability: Good — clear comments reference rule numbers (Rule 3, 6, 7, 8), helper functions well-named
- Issues: None

BLOCKING ISSUES:
- None

NON-BLOCKING NOTES:
- The `var sm task.StateMachine` instantiation is repeated in each handler. Since StateMachine is stateless, a package-level var or shared helper could reduce boilerplate, but the current approach is explicit and idiomatic Go. Not a concern.
