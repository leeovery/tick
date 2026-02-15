package scripts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/testutil"
	"gopkg.in/yaml.v3"
)

// workflow represents the subset of a GitHub Actions workflow file
// that we need to validate for the release pipeline.
type workflow struct {
	Name        string `yaml:"name"`
	Permissions struct {
		Contents string `yaml:"contents"`
	} `yaml:"permissions"`
	On struct {
		Push struct {
			Tags     []string `yaml:"tags"`
			Branches []string `yaml:"branches"`
		} `yaml:"push"`
		PullRequest *struct{} `yaml:"pull_request"`
	} `yaml:"on"`
	Jobs map[string]job `yaml:"jobs"`
}

type job struct {
	RunsOn string `yaml:"runs-on"`
	Steps  []step `yaml:"steps"`
}

type step struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	With map[string]string `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
	ID   string            `yaml:"id"`
}

// loadWorkflow parses the release.yml workflow file and returns the parsed structure.
func loadWorkflow(t *testing.T) workflow {
	t.Helper()

	root := testutil.FindRepoRoot(t)
	path := filepath.Join(root, ".github", "workflows", "release.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read release.yml: %v", err)
	}

	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatalf("cannot parse release.yml: %v", err)
	}
	return w
}

// matchesGitHubActionsPattern checks whether a tag matches a GitHub Actions
// filter pattern. GitHub Actions uses a custom glob-like syntax:
//   - * matches any character except /
//   - ** matches any character including /
//   - ? matches a single character
//   - [abc] matches any character in the set
//   - [a-z] matches any character in the range
//
// Note: + is NOT special in GitHub Actions patterns (it is literal).
func matchesGitHubActionsPattern(pattern, tag string) bool {
	return matchPattern(pattern, tag, 0, 0)
}

func matchPattern(pattern, text string, pi, ti int) bool {
	for pi < len(pattern) && ti < len(text) {
		switch {
		case pattern[pi] == '*':
			if pi+1 < len(pattern) && pattern[pi+1] == '*' {
				// ** matches everything including /
				pi += 2
				// Try matching rest from every position.
				for k := ti; k <= len(text); k++ {
					if matchPattern(pattern, text, pi, k) {
						return true
					}
				}
				return false
			}
			// * matches everything except /
			pi++
			for k := ti; k <= len(text); k++ {
				if k > ti && text[k-1] == '/' {
					break
				}
				if matchPattern(pattern, text, pi, k) {
					return true
				}
			}
			return false
		case pattern[pi] == '?':
			if text[ti] == '/' {
				return false
			}
			pi++
			ti++
		case pattern[pi] == '[':
			// Character class.
			pi++ // skip [
			negate := false
			if pi < len(pattern) && pattern[pi] == '!' {
				negate = true
				pi++
			}
			matched := false
			for pi < len(pattern) && pattern[pi] != ']' {
				lo := pattern[pi]
				pi++
				if pi+1 < len(pattern) && pattern[pi] == '-' {
					pi++ // skip -
					hi := pattern[pi]
					pi++
					if text[ti] >= lo && text[ti] <= hi {
						matched = true
					}
				} else {
					if text[ti] == lo {
						matched = true
					}
				}
			}
			if pi < len(pattern) {
				pi++ // skip ]
			}
			if matched == negate {
				return false
			}
			ti++
		default:
			if pattern[pi] != text[ti] {
				return false
			}
			pi++
			ti++
		}
	}
	// Handle trailing stars.
	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}
	return pi == len(pattern) && ti == len(text)
}

func TestReleaseWorkflow(t *testing.T) {
	w := loadWorkflow(t)

	t.Run("workflow triggers on valid semver tag v1.0.0", func(t *testing.T) {
		assertTagMatches(t, w.On.Push.Tags, "v1.0.0", true)
	})

	t.Run("workflow triggers on valid semver tag v0.1.0", func(t *testing.T) {
		assertTagMatches(t, w.On.Push.Tags, "v0.1.0", true)
	})

	t.Run("workflow triggers on valid semver tag v12.34.56", func(t *testing.T) {
		assertTagMatches(t, w.On.Push.Tags, "v12.34.56", true)
	})

	t.Run("workflow does not trigger on pre-release tag v1.0.0-beta", func(t *testing.T) {
		assertTagMatches(t, w.On.Push.Tags, "v1.0.0-beta", false)
	})

	t.Run("workflow does not trigger on pre-release tag v1.0.0-rc.1", func(t *testing.T) {
		assertTagMatches(t, w.On.Push.Tags, "v1.0.0-rc.1", false)
	})

	t.Run("workflow does not trigger on non-version tag latest", func(t *testing.T) {
		assertTagMatches(t, w.On.Push.Tags, "latest", false)
	})

	t.Run("workflow does not trigger on branch push to main", func(t *testing.T) {
		if len(w.On.Push.Branches) > 0 {
			t.Errorf("expected no branch triggers, got: %v", w.On.Push.Branches)
		}
	})

	t.Run("workflow does not trigger on pull requests", func(t *testing.T) {
		if w.On.PullRequest != nil {
			t.Error("expected no pull_request trigger")
		}
	})

	t.Run("workflow does not trigger on tag without v prefix like 1.0.0", func(t *testing.T) {
		assertTagMatches(t, w.On.Push.Tags, "1.0.0", false)
	})

	t.Run("checkout step uses fetch-depth 0 for full history", func(t *testing.T) {
		s, ok := findStepByUses(w, "actions/checkout")
		if !ok || s.With["fetch-depth"] != "0" {
			t.Error("checkout step must use fetch-depth: 0")
		}
	})

	t.Run("goreleaser step has GITHUB_TOKEN configured", func(t *testing.T) {
		s, ok := findStepByUses(w, "goreleaser/goreleaser-action")
		if !ok || s.Env["GITHUB_TOKEN"] != "${{ secrets.GITHUB_TOKEN }}" {
			t.Error("goreleaser step must have GITHUB_TOKEN set to ${{ secrets.GITHUB_TOKEN }}")
		}
	})

	t.Run("workflow has contents write permission", func(t *testing.T) {
		if w.Permissions.Contents != "write" {
			t.Errorf("expected permissions.contents to be 'write', got %q", w.Permissions.Contents)
		}
	})

	t.Run("go setup uses go-version-file go.mod", func(t *testing.T) {
		s, ok := findStepByUses(w, "actions/setup-go")
		if !ok || s.With["go-version-file"] != "go.mod" {
			t.Error("setup-go step must use go-version-file: go.mod")
		}
	})

	t.Run("workflow runs on ubuntu-latest", func(t *testing.T) {
		for _, j := range w.Jobs {
			if j.RunsOn == "ubuntu-latest" {
				return
			}
		}
		t.Error("expected at least one job to run on ubuntu-latest")
	})

	t.Run("goreleaser invoked with release --clean", func(t *testing.T) {
		s, ok := findStepByUses(w, "goreleaser/goreleaser-action")
		if !ok || s.With["args"] != "release --clean" {
			t.Error("goreleaser step must use args: release --clean")
		}
	})
}

// findStepByUses searches all jobs in the workflow for a step whose Uses field
// contains usesSubstring. Returns the first matching step and true, or a zero
// step and false if none matched.
func findStepByUses(w workflow, usesSubstring string) (step, bool) {
	for _, j := range w.Jobs {
		for _, s := range j.Steps {
			if strings.Contains(s.Uses, usesSubstring) {
				return s, true
			}
		}
	}
	return step{}, false
}

// findStepByName searches all jobs for a step whose Name contains nameSubstring.
func findStepByName(w workflow, nameSubstring string) (step, bool) {
	for _, j := range w.Jobs {
		for _, s := range j.Steps {
			if strings.Contains(s.Name, nameSubstring) {
				return s, true
			}
		}
	}
	return step{}, false
}

func TestReleaseWorkflowHomebrewDispatch(t *testing.T) {
	w := loadWorkflow(t)

	t.Run("workflow has checksum extraction step", func(t *testing.T) {
		s, ok := findStepByName(w, "Extract checksums")
		if !ok {
			t.Fatal("expected an 'Extract checksums' step in the workflow")
		}
		if s.ID != "checksums" {
			t.Errorf("expected checksum step id 'checksums', got %q", s.ID)
		}
		if !strings.Contains(s.Run, "checksums.txt") {
			t.Error("checksum step must reference checksums.txt")
		}
		if !strings.Contains(s.Run, "darwin_arm64") {
			t.Error("checksum step must extract darwin_arm64 hash")
		}
		if !strings.Contains(s.Run, "darwin_amd64") {
			t.Error("checksum step must extract darwin_amd64 hash")
		}
	})

	t.Run("workflow has homebrew dispatch step", func(t *testing.T) {
		s, ok := findStepByName(w, "Dispatch to homebrew-tools")
		if !ok {
			t.Fatal("expected a 'Dispatch to homebrew-tools' step in the workflow")
		}
		if !strings.Contains(s.Run, "homebrew-tools") {
			t.Error("dispatch step must target homebrew-tools repo")
		}
		if !strings.Contains(s.Run, "update-formula") {
			t.Error("dispatch step must use update-formula event type")
		}
		if !strings.Contains(s.Run, `"tool": "tick"`) {
			t.Error("dispatch step must include tool=tick in payload")
		}
		if !strings.Contains(s.Run, "sha256_arm64") {
			t.Error("dispatch step must include sha256_arm64 in payload")
		}
		if !strings.Contains(s.Run, "sha256_amd64") {
			t.Error("dispatch step must include sha256_amd64 in payload")
		}
		if !strings.Contains(s.Run, "CICD_PAT") {
			t.Error("dispatch step must use CICD_PAT secret")
		}
	})
}

// assertTagMatches checks whether the workflow tag patterns match (or don't match) the given tag.
// GitHub Actions evaluates filter patterns in order: positive patterns include, negative
// patterns (prefixed with !) exclude. A tag matches if it matches at least one positive
// pattern and is not excluded by any negative pattern.
func assertTagMatches(t *testing.T, patterns []string, tag string, shouldMatch bool) {
	t.Helper()
	if len(patterns) == 0 {
		t.Fatal("workflow has no tag patterns configured")
	}

	included := false
	excluded := false
	for _, p := range patterns {
		if len(p) > 0 && p[0] == '!' {
			// Negative pattern -- excludes matching tags.
			if matchesGitHubActionsPattern(p[1:], tag) {
				excluded = true
			}
		} else {
			// Positive pattern -- includes matching tags.
			if matchesGitHubActionsPattern(p, tag) {
				included = true
			}
		}
	}

	matched := included && !excluded
	if shouldMatch && !matched {
		t.Errorf("expected tag %q to match patterns %v, but it did not", tag, patterns)
	}
	if !shouldMatch && matched {
		t.Errorf("expected tag %q to NOT match patterns %v, but it did", tag, patterns)
	}
}
