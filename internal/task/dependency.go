package task

// ValidateDependency checks that adding newBlockedByID as a blocker of taskID
// does not create a circular dependency or a child-blocked-by-parent relationship.
// It delegates to StateMachine.ValidateAddDep for the actual validation logic.
func ValidateDependency(tasks []Task, taskID, newBlockedByID string) error {
	var sm StateMachine
	return sm.ValidateAddDep(tasks, taskID, newBlockedByID)
}

// ValidateDependencies validates multiple blocked_by IDs for a single task,
// checking each sequentially and failing on the first error.
func ValidateDependencies(tasks []Task, taskID string, blockedByIDs []string) error {
	for _, blockedByID := range blockedByIDs {
		if err := ValidateDependency(tasks, taskID, blockedByID); err != nil {
			return err
		}
	}
	return nil
}
