package migrate

// DryRunTaskCreator implements TaskCreator as a no-op. It always returns an
// empty ID and nil error, allowing the migration engine to run without
// persisting any data. This is used when --dry-run is set.
type DryRunTaskCreator struct{}

// Compile-time check that DryRunTaskCreator satisfies TaskCreator.
var _ TaskCreator = (*DryRunTaskCreator)(nil)

// CreateTask returns an empty string and nil error, performing no persistence.
func (d *DryRunTaskCreator) CreateTask(_ MigratedTask) (string, error) {
	return "", nil
}
