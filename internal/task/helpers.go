package task

// deduplicateStrings normalizes each item using the provided function, filters empties,
// and returns unique items in first-occurrence order.
func deduplicateStrings(items []string, normalize func(string) string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		normalized := normalize(item)
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
