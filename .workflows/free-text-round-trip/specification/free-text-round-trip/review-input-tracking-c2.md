# Review Tracking: Free Text Round Trip - Input Review

## Findings

### 1. The route for correcting the two older specifications loses the step that makes the correction stick

**Source**: `discussion/free-text-round-trip.md` — "Published Documentation Owed a Correction" → Decision: "**The specifications are corrected selectively, by judgement, through the corrigendum facility — not as a blanket rewrite.**"
**Category**: Enhancement to existing topic
**Move**: settled
**Affects**: §12.2 Two completed specifications are corrected selectively

**Problem**:
Two older specifications state things this work replaces — that long text fields get their own unstructured sections, and that the arrow-and-`(auto)` lines are the machine-readable cascade form. Both are amended, but the amendment is described only as "present the change, confirm, edit". A specification's content stays live in the knowledge base at full confidence, so an edit that stops at the file leaves the superseded claims still being served as validated context to every later question asked of the knowledge base — including questions asked while building this work. The correction would look done and change nothing that anyone actually reads. The source names the mechanism that closes this — the corrigendum facility, whose defining steps are the dated entry recording what the document used to claim and the re-index that replaces the stale content — and the specification carries only the confirmation gate, referring to the facility obliquely as "the facility for amending them".

**Proposal**:
Name the facility the source named and carry the two steps that make the amendment take effect: the dated corrigendum entry and the re-index. The source decided the route ("through the corrigendum facility"); the specification narrowed it to one of its steps.

**Current**:
Both belong to completed work units, so the correcting route is the one that presents each proposed amendment and confirms before editing another unit's record.

**Proposed Text**:
Both belong to completed work units, so the correction runs through the corrigendum facility: each proposed amendment is presented and confirmed before another unit's record is edited, the wrong claim is replaced in place, a dated corrigendum entry records what the document used to claim and what is true instead, and the document is re-indexed. The re-index is the point of the route — a specification's content stays live in the knowledge base at full confidence, so an amendment that stops at the file leaves the superseded claim being served as validated context to every later query.

**Resolution**: Approved
**Notes**: Applied verbatim under auto. Disposal verified the facility's own steps before applying — correcting-historical-artifacts.md defines edit-in-place, a dated `## Corrigenda` entry, and an idempotent re-index — so the text describes a real route rather than an invented one.

---
