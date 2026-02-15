package migrate

import (
	"errors"
	"testing"
)

func TestUnknownProviderError(t *testing.T) {
	t.Run("Error includes the unknown provider name in quotes", func(t *testing.T) {
		err := &UnknownProviderError{
			Name:      "xyz",
			Available: []string{"beads"},
		}
		got := err.Error()
		want := "Unknown provider \"xyz\"\n\nAvailable providers:\n  - beads"
		if got != want {
			t.Errorf("Error() =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("Error includes Available providers header", func(t *testing.T) {
		err := &UnknownProviderError{
			Name:      "jira",
			Available: []string{"beads"},
		}
		got := err.Error()
		want := "Unknown provider \"jira\"\n\nAvailable providers:\n  - beads"
		if got != want {
			t.Errorf("Error() =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("Error lists each registered provider with indent dash prefix", func(t *testing.T) {
		err := &UnknownProviderError{
			Name:      "xyz",
			Available: []string{"beads", "jira", "linear"},
		}
		got := err.Error()
		want := "Unknown provider \"xyz\"\n\nAvailable providers:\n  - beads\n  - jira\n  - linear"
		if got != want {
			t.Errorf("Error() =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("Error includes empty line between error line and available list", func(t *testing.T) {
		err := &UnknownProviderError{
			Name:      "xyz",
			Available: []string{"beads"},
		}
		got := err.Error()
		// The format should be: line1\n\nline3\n...
		// i.e., "Unknown provider \"xyz\"\n\nAvailable providers:\n  - beads"
		want := "Unknown provider \"xyz\"\n\nAvailable providers:\n  - beads"
		if got != want {
			t.Errorf("Error() =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("single provider in registry: error message lists one provider", func(t *testing.T) {
		err := &UnknownProviderError{
			Name:      "xyz",
			Available: []string{"beads"},
		}
		got := err.Error()
		want := "Unknown provider \"xyz\"\n\nAvailable providers:\n  - beads"
		if got != want {
			t.Errorf("Error() =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("multiple providers in registry: error message lists all providers alphabetically", func(t *testing.T) {
		err := &UnknownProviderError{
			Name:      "xyz",
			Available: []string{"linear", "beads", "jira"},
		}
		got := err.Error()
		// The Error() method should sort the available list
		want := "Unknown provider \"xyz\"\n\nAvailable providers:\n  - beads\n  - jira\n  - linear"
		if got != want {
			t.Errorf("Error() =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("UnknownProviderError is type-assertable via errors.As", func(t *testing.T) {
		var original error = &UnknownProviderError{
			Name:      "xyz",
			Available: []string{"beads"},
		}
		var target *UnknownProviderError
		if !errors.As(original, &target) {
			t.Fatal("errors.As failed to match *UnknownProviderError")
		}
		if target.Name != "xyz" {
			t.Errorf("target.Name = %q, want %q", target.Name, "xyz")
		}
		if len(target.Available) != 1 || target.Available[0] != "beads" {
			t.Errorf("target.Available = %v, want [beads]", target.Available)
		}
	})
}
