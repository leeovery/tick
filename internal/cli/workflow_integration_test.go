package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestWorkflowIntegration exercises the primary agent workflow end-to-end:
// init -> create tasks with dependencies/hierarchy -> ready (verify correct tasks)
// -> transition -> ready (verify unblocking) -> complete all -> stats.
func TestWorkflowIntegration(t *testing.T) {
	// Step 1: Initialize a tick project.
	dir, _ := setupTickProject(t)

	// Step 2: Create a parent/epic task.
	stdout, stderr, exitCode := runCreate(t, dir, "--quiet", "Build authentication system")
	if exitCode != 0 {
		t.Fatalf("create epic failed: exit=%d stderr=%q", exitCode, stderr)
	}
	epicID := strings.TrimSpace(stdout)

	// Step 3: Create child tasks under the epic.
	// Child A: "Setup database schema" - no dependencies, should be ready immediately.
	stdout, stderr, exitCode = runCreate(t, dir, "--quiet", "Setup database schema", "--parent", epicID)
	if exitCode != 0 {
		t.Fatalf("create child A failed: exit=%d stderr=%q", exitCode, stderr)
	}
	childA := strings.TrimSpace(stdout)

	// Child B: "Implement login endpoint" - blocked by child A.
	stdout, stderr, exitCode = runCreate(t, dir, "--quiet", "Implement login endpoint", "--parent", epicID, "--blocked-by", childA)
	if exitCode != 0 {
		t.Fatalf("create child B failed: exit=%d stderr=%q", exitCode, stderr)
	}
	childB := strings.TrimSpace(stdout)

	// Child C: "Add session management" - blocked by child B (chain: A -> B -> C).
	stdout, stderr, exitCode = runCreate(t, dir, "--quiet", "Add session management", "--parent", epicID, "--blocked-by", childB)
	if exitCode != 0 {
		t.Fatalf("create child C failed: exit=%d stderr=%q", exitCode, stderr)
	}
	childC := strings.TrimSpace(stdout)

	// Child D: "Write auth tests" - blocked by both child B and child C.
	stdout, stderr, exitCode = runCreate(t, dir, "--quiet", "Write auth tests", "--parent", epicID, "--blocked-by", childB+","+childC)
	if exitCode != 0 {
		t.Fatalf("create child D failed: exit=%d stderr=%q", exitCode, stderr)
	}
	childD := strings.TrimSpace(stdout)

	// Step 4: Verify initial ready set.
	// Only child A should be ready (no blockers, leaf task).
	// The epic should NOT be ready (has open children).
	// Children B, C, D should NOT be ready (blocked).
	stdout, stderr, exitCode = runReady(t, dir, "--quiet")
	if exitCode != 0 {
		t.Fatalf("ready (step 4) failed: exit=%d stderr=%q", exitCode, stderr)
	}
	readyIDs := parseQuietIDs(stdout)

	assertContains(t, readyIDs, childA, "child A should be ready (no blockers)")
	assertNotContains(t, readyIDs, epicID, "epic should NOT be ready (has open children)")
	assertNotContains(t, readyIDs, childB, "child B should NOT be ready (blocked by A)")
	assertNotContains(t, readyIDs, childC, "child C should NOT be ready (blocked by B)")
	assertNotContains(t, readyIDs, childD, "child D should NOT be ready (blocked by B and C)")

	// Step 5: Start and complete child A (the blocker).
	_, stderr, exitCode = runTransition(t, dir, "start", childA)
	if exitCode != 0 {
		t.Fatalf("start child A failed: exit=%d stderr=%q", exitCode, stderr)
	}

	// While child A is in_progress, child B should still be blocked.
	stdout, stderr, exitCode = runReady(t, dir, "--quiet")
	if exitCode != 0 {
		t.Fatalf("ready (A in_progress) failed: exit=%d stderr=%q", exitCode, stderr)
	}
	readyIDs = parseQuietIDs(stdout)
	assertNotContains(t, readyIDs, childA, "child A should NOT be ready (in_progress)")
	assertNotContains(t, readyIDs, childB, "child B should NOT be ready (A still in_progress)")

	// Complete child A.
	_, stderr, exitCode = runTransition(t, dir, "done", childA)
	if exitCode != 0 {
		t.Fatalf("done child A failed: exit=%d stderr=%q", exitCode, stderr)
	}

	// Step 6: Verify unblocking - child B should now be ready.
	stdout, stderr, exitCode = runReady(t, dir, "--quiet")
	if exitCode != 0 {
		t.Fatalf("ready (step 6) failed: exit=%d stderr=%q", exitCode, stderr)
	}
	readyIDs = parseQuietIDs(stdout)

	assertContains(t, readyIDs, childB, "child B should be ready (A is done)")
	assertNotContains(t, readyIDs, childA, "child A should NOT be ready (done)")
	assertNotContains(t, readyIDs, epicID, "epic should NOT be ready (has open children)")
	assertNotContains(t, readyIDs, childC, "child C should NOT be ready (blocked by B)")
	assertNotContains(t, readyIDs, childD, "child D should NOT be ready (blocked by B and C)")

	// Step 7: Complete child B - this should unblock child C.
	_, stderr, exitCode = runTransition(t, dir, "done", childB)
	if exitCode != 0 {
		t.Fatalf("done child B failed: exit=%d stderr=%q", exitCode, stderr)
	}

	stdout, stderr, exitCode = runReady(t, dir, "--quiet")
	if exitCode != 0 {
		t.Fatalf("ready (after B done) failed: exit=%d stderr=%q", exitCode, stderr)
	}
	readyIDs = parseQuietIDs(stdout)

	assertContains(t, readyIDs, childC, "child C should be ready (B is done)")
	assertNotContains(t, readyIDs, childD, "child D should NOT be ready (C still open)")
	assertNotContains(t, readyIDs, epicID, "epic should NOT be ready (has open children)")

	// Step 8: Complete child C - this should unblock child D.
	_, stderr, exitCode = runTransition(t, dir, "done", childC)
	if exitCode != 0 {
		t.Fatalf("done child C failed: exit=%d stderr=%q", exitCode, stderr)
	}

	stdout, stderr, exitCode = runReady(t, dir, "--quiet")
	if exitCode != 0 {
		t.Fatalf("ready (after C done) failed: exit=%d stderr=%q", exitCode, stderr)
	}
	readyIDs = parseQuietIDs(stdout)

	assertContains(t, readyIDs, childD, "child D should be ready (B and C are done)")
	assertNotContains(t, readyIDs, epicID, "epic should NOT be ready (child D still open)")

	// Step 9: Complete child D - all children now closed, epic should become ready.
	_, stderr, exitCode = runTransition(t, dir, "done", childD)
	if exitCode != 0 {
		t.Fatalf("done child D failed: exit=%d stderr=%q", exitCode, stderr)
	}

	stdout, stderr, exitCode = runReady(t, dir, "--quiet")
	if exitCode != 0 {
		t.Fatalf("ready (after D done) failed: exit=%d stderr=%q", exitCode, stderr)
	}
	readyIDs = parseQuietIDs(stdout)

	assertContains(t, readyIDs, epicID, "epic should be ready (all children done)")

	// Step 10: Complete the epic.
	_, stderr, exitCode = runTransition(t, dir, "done", epicID)
	if exitCode != 0 {
		t.Fatalf("done epic failed: exit=%d stderr=%q", exitCode, stderr)
	}

	// No tasks should be ready now.
	stdout, stderr, exitCode = runReady(t, dir)
	if exitCode != 0 {
		t.Fatalf("ready (final) failed: exit=%d stderr=%q", exitCode, stderr)
	}
	if stdout != "No tasks found.\n" {
		t.Errorf("expected no ready tasks after all done, got %q", stdout)
	}

	// Step 11: Verify stats reflect the final state.
	stdout, stderr, exitCode = runStats(t, dir, "--json")
	if exitCode != 0 {
		t.Fatalf("stats failed: exit=%d stderr=%q", exitCode, stderr)
	}

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &stats); err != nil {
		t.Fatalf("invalid stats JSON: %v\nstdout: %s", err, stdout)
	}

	// 5 total tasks: epic + 4 children, all done.
	if stats["total"] != float64(5) {
		t.Errorf("stats total = %v, want 5", stats["total"])
	}

	byStatus := stats["by_status"].(map[string]interface{})
	if byStatus["done"] != float64(5) {
		t.Errorf("stats done = %v, want 5", byStatus["done"])
	}
	if byStatus["open"] != float64(0) {
		t.Errorf("stats open = %v, want 0", byStatus["open"])
	}
	if byStatus["in_progress"] != float64(0) {
		t.Errorf("stats in_progress = %v, want 0", byStatus["in_progress"])
	}
	if byStatus["cancelled"] != float64(0) {
		t.Errorf("stats cancelled = %v, want 0", byStatus["cancelled"])
	}

	workflow := stats["workflow"].(map[string]interface{})
	if workflow["ready"] != float64(0) {
		t.Errorf("stats ready = %v, want 0", workflow["ready"])
	}
	if workflow["blocked"] != float64(0) {
		t.Errorf("stats blocked = %v, want 0", workflow["blocked"])
	}
}

// parseQuietIDs splits --quiet output into a slice of task IDs.
func parseQuietIDs(stdout string) []string {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// assertContains checks that the ID appears in the list.
func assertContains(t *testing.T, ids []string, id string, msg string) {
	t.Helper()
	for _, v := range ids {
		if v == id {
			return
		}
	}
	t.Errorf("%s: %s not found in %v", msg, id, ids)
}

// assertNotContains checks that the ID does NOT appear in the list.
func assertNotContains(t *testing.T, ids []string, id string, msg string) {
	t.Helper()
	for _, v := range ids {
		if v == id {
			t.Errorf("%s: %s should not be in %v", msg, id, ids)
			return
		}
	}
}
