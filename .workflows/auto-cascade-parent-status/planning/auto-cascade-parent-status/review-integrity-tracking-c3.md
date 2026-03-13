---
status: complete
created: 2026-03-06
cycle: 3
phase: Plan Integrity Review
topic: Auto-Cascade Parent Status
---

# Review Tracking: Auto-Cascade Parent Status - Integrity

## Findings

### 1. auto-cascade-parent-status-3-5 misstates Rule 3 condition and lacks self-contained decision logic

**Severity**: Important
**Plan Reference**: Phase 3 / auto-cascade-parent-status-3-5 (tick-2bf0f6)
**Category**: Task Self-Containment / Acceptance Criteria Quality
**Change Type**: update-task

**Details**:
The acceptance criterion "If all remaining children of original parent done, original parent auto-completes to done" misstates Rule 3. The spec says the parent goes to `done` when "at least one child is done" among all-terminal children -- not when ALL are done. The mixed case (some done, some cancelled) should also produce `done`, but the current AC implies all must be done. An implementer could incorrectly require all children to be `done` before auto-completing to `done`.

Additionally, the Do section says "evaluate Rule 3 on original parent -- check if all remaining children terminal, if so call ApplyWithCascades done/cancel" without specifying how to choose between done and cancel. This forces the implementer to look up Rule 3 elsewhere. The task should contain the decision logic inline.

**Current**:
```
Do:
- Capture original parent ID before updating
- If new parent not empty: ValidateAddChild (blocks cancelled), if done call ApplyWithCascades reopen (Rule 6)
- Update parent field
- If origParent != "" and changed: evaluate Rule 3 on original parent -- check if all remaining children terminal, if so call ApplyWithCascades done/cancel
- After Mutate: output updated task detail, then cascade output

Acceptance Criteria:
- [ ] Reparenting to cancelled parent returns error
- [ ] Reparenting to done parent triggers reopen cascade (Rule 6)
- [ ] Reparenting away triggers Rule 3 re-evaluation on original parent
- [ ] If all remaining children of original parent done, original parent auto-completes to done
- [ ] If all remaining children cancelled, original parent auto-completes to cancelled
- [ ] Clearing parent (--parent "") triggers Rule 3 on original parent
- [ ] All changes atomic in single Mutate
- [ ] Existing update tests pass
```

**Proposed**:
```
Do:
- Capture original parent ID before updating
- If new parent not empty: ValidateAddChild (blocks cancelled), if done call ApplyWithCascades reopen (Rule 6)
- Update parent field
- If origParent != "" and changed: evaluate Rule 3 on original parent -- gather all remaining children of original parent, check if all are terminal (done or cancelled). If so, determine action: if at least one child is done, use "done"; if all children are cancelled, use "cancel". Call ApplyWithCascades(tasks, originalParent, action) to transition and cascade upward.
- After Mutate: output updated task detail, then cascade output

Acceptance Criteria:
- [ ] Reparenting to cancelled parent returns error
- [ ] Reparenting to done parent triggers reopen cascade (Rule 6)
- [ ] Reparenting away triggers Rule 3 re-evaluation on original parent
- [ ] If all remaining children terminal and at least one is done, original parent auto-completes to done
- [ ] If all remaining children cancelled (none done), original parent auto-completes to cancelled
- [ ] Mixed done and cancelled remaining children: original parent auto-completes to done
- [ ] Non-terminal remaining children prevent auto-completion
- [ ] Clearing parent (--parent "") triggers Rule 3 on original parent
- [ ] All changes atomic in single Mutate
- [ ] Existing update tests pass
```

**Resolution**: Fixed
**Notes**: Do step and ACs corrected with inline decision logic and mixed-state criterion.

---

### 2. auto-cascade-parent-status-3-5 missing test for mixed done/cancelled children on Rule 3 evaluation

**Severity**: Important
**Plan Reference**: Phase 3 / auto-cascade-parent-status-3-5 (tick-2bf0f6)
**Category**: Acceptance Criteria Quality
**Change Type**: update-task

**Details**:
The test list for auto-cascade-parent-status-3-5 lacks coverage for the case where the original parent has a mix of done and cancelled remaining children after reparenting away. This is the key differentiator of Rule 3's "at least one done" logic and should be explicitly tested at the CLI wiring level to catch misimplementation.

**Current**:
```
Tests:
- "it blocks reparenting to cancelled parent"
- "it reopens done parent when reparenting to it"
- "it triggers Rule 3 on original parent when reparenting away"
- "it triggers Rule 3 with cancelled result"
- "it does not trigger Rule 3 when original parent still has non-terminal children"
- "it handles clearing parent"
- "it handles reparent to done parent plus Rule 3 on original"
```

**Proposed**:
```
Tests:
- "it blocks reparenting to cancelled parent"
- "it reopens done parent when reparenting to it"
- "it triggers Rule 3 on original parent when reparenting away"
- "it triggers Rule 3 with cancelled result when all remaining children cancelled"
- "it triggers Rule 3 with done result when remaining children are mix of done and cancelled"
- "it does not trigger Rule 3 when original parent still has non-terminal children"
- "it handles clearing parent"
- "it handles reparent to done parent plus Rule 3 on original"
```

**Resolution**: Fixed
**Notes**: Do step and ACs corrected with inline decision logic and mixed-state criterion. Renamed existing cancelled test for clarity and added mixed-state test.

---
