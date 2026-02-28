package task

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	maxTagLength   = 30
	maxTagsPerTask = 10
)

// tagPattern matches strict kebab-case: one or more segments of lowercase alphanumeric
// characters separated by single hyphens.
var tagPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// NormalizeTag trims whitespace and lowercases a tag string.
func NormalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// ValidateTag checks that a single tag is non-empty, matches kebab-case, and is at most 30 chars.
func ValidateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if len(tag) > maxTagLength {
		return fmt.Errorf("tag %q exceeds maximum length of %d characters", tag, maxTagLength)
	}
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("tag %q must be kebab-case (lowercase alphanumeric segments separated by single hyphens)", tag)
	}
	return nil
}

// DeduplicateTags normalizes tags, filters empties, and returns unique tags in first-occurrence order.
func DeduplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, tag := range tags {
		normalized := NormalizeTag(tag)
		if normalized == "" {
			continue
		}
		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}
	return result
}

// ValidateTags normalizes, filters empties, deduplicates, validates each tag, and checks count <= 10.
func ValidateTags(tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	deduped := DeduplicateTags(tags)

	for _, tag := range deduped {
		if err := ValidateTag(tag); err != nil {
			return err
		}
	}

	if len(deduped) > maxTagsPerTask {
		return fmt.Errorf("too many tags: %d exceeds maximum of %d per task", len(deduped), maxTagsPerTask)
	}

	return nil
}
