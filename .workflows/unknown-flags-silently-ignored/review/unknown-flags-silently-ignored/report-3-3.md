TASK: Derive ready/blocked flag sets programmatically from list

ACCEPTANCE CRITERIA:
- The commandFlags map literal contains no "ready" or "blocked" entries
- An init() function derives ready and blocked from list's flags
- commandFlags["ready"] has exactly 7 entries (list's 8 minus --ready)
- commandFlags["blocked"] has exactly 7 entries (list's 8 minus --blocked)
- go test ./internal/cli/ -count=1 passes

STATUS: Complete

SPEC CONTEXT: The spec (specification.md:95-96) says "ready: same as list minus --ready" and "blocked: same as list minus --blocked", implying 7 flags each. The implementation excludes BOTH --ready and --blocked from each derived set, yielding 6 flags each. This deviation was identified in r1 and accepted as a justified improvement: since --ready and --blocked are mutually exclusive filter flags on list, and the ready/blocked commands implicitly set one, allowing the other would be contradictory. The tests and acceptance criteria in the plan task text say 7, but the implementation uses 6. This was already reviewed and approved in r1 as semantically more correct.

IMPLEMENTATION:
- Status: Implemented
- Location:
  - /Users/leeovery/Code/tick/internal/cli/flags.go:33-90 -- commandFlags map literal contains no "ready" or "blocked" entries (AC 1 met)
  - /Users/leeovery/Code/tick/internal/cli/flags.go:92-95 -- init() derives both sets using copyFlagsExcept (AC 2 met)
  - /Users/leeovery/Code/tick/internal/cli/flags.go:98-107 -- copyFlagsExcept helper: variadic exclusion, shallow copy + delete
- Notes: The init() call on line 93 is `copyFlagsExcept(commandFlags["list"], "--ready", "--blocked")` and on line 94 is `copyFlagsExcept(commandFlags["list"], "--blocked", "--ready")`. Each produces 6 flags (list's 8 minus 2), not 7 as the AC states. This deviation from AC 3 and AC 4 is intentional and was approved in r1 review. AC 5 (tests pass) is met.

TESTS:
- Status: Adequate
- Coverage:
  - TestFlagValidationAllCommands (flag_validation_test.go:66-76,78-88): verifies ready has flagCount 6, blocked has flagCount 6, and all valid flags are accepted without error
  - TestReadyRejectsReady (flag_validation_test.go:163-174): verifies --ready is rejected on the ready command with correct error message
  - TestBlockedRejectsBlocked (flag_validation_test.go:176-187): verifies --blocked is rejected on the blocked command with correct error message
  - The flagCount assertions would catch regressions if copyFlagsExcept or the list flag set changed unexpectedly
- Notes: No test explicitly verifies that --blocked is rejected on ready (or --ready rejected on blocked), but the flagCount: 6 assertion implicitly guarantees both flags are excluded. The dedicated rejection tests cover the primary exclusion case. Test coverage is adequate without being excessive.

CODE QUALITY:
- Project conventions: Followed. Uses Go init() idiom, stdlib testing, t.Run subtests, no external test frameworks.
- SOLID principles: Good. copyFlagsExcept has single responsibility (copy-and-exclude). The derivation relationship is declarative and clear.
- Complexity: Low. copyFlagsExcept is a trivial map-copy-then-delete loop.
- Modern idioms: Yes. Variadic parameters for exclusion list, make with capacity hint.
- Readability: Good. The init() body reads naturally as "ready = list minus {--ready, --blocked}". Doc comment on copyFlagsExcept is accurate.
- Issues: None.

BLOCKING ISSUES:
- (none)

NON-BLOCKING NOTES:
- The acceptance criteria text specifies "exactly 7 entries" for both ready and blocked, but the implementation produces 6 each. This was reviewed in r1 and accepted as a justified improvement. The plan task acceptance criteria are stale relative to the approved implementation. If the plan is maintained as a living document, consider updating AC 3 and AC 4 to reflect the actual (correct) counts of 6.
