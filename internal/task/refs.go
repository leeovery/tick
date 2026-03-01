package task

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const (
	maxRefLength   = 200
	maxRefsPerTask = 10
)

// ValidateRef checks that a single ref is non-empty after trimming, contains no commas
// or whitespace, and is at most 200 characters.
func ValidateRef(ref string) error {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return errors.New("ref cannot be empty")
	}
	if strings.ContainsRune(trimmed, ',') {
		return fmt.Errorf("ref %q must not contain commas", trimmed)
	}
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
			return fmt.Errorf("ref %q must not contain whitespace", trimmed)
		}
	}
	if len(trimmed) > maxRefLength {
		return fmt.Errorf("ref exceeds maximum length of %d characters", maxRefLength)
	}
	return nil
}

// DeduplicateRefs trims and deduplicates refs, preserving first-occurrence order.
// Empty refs after trimming are filtered out.
func DeduplicateRefs(refs []string) []string {
	return deduplicateStrings(refs, strings.TrimSpace)
}

// ValidateRefs trims, deduplicates, validates each ref, and checks count <= 10.
func ValidateRefs(refs []string) error {
	if len(refs) == 0 {
		return nil
	}

	deduped := DeduplicateRefs(refs)

	for _, ref := range deduped {
		if err := ValidateRef(ref); err != nil {
			return err
		}
	}

	if len(deduped) > maxRefsPerTask {
		return fmt.Errorf("too many refs: %d exceeds maximum of %d per task", len(deduped), maxRefsPerTask)
	}

	return nil
}

// ParseRefs splits a comma-separated input string into refs, trims each, deduplicates,
// and validates the result.
func ParseRefs(input string) ([]string, error) {
	if strings.TrimSpace(input) == "" {
		return nil, errors.New("refs input cannot be empty")
	}

	parts := strings.Split(input, ",")
	refs := DeduplicateRefs(parts)

	for _, ref := range refs {
		if err := ValidateRef(ref); err != nil {
			return nil, err
		}
	}

	if len(refs) > maxRefsPerTask {
		return nil, fmt.Errorf("too many refs: %d exceeds maximum of %d per task", len(refs), maxRefsPerTask)
	}

	return refs, nil
}
