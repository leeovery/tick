package scripts_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/testutil"
	"gopkg.in/yaml.v3"
)

// goreleaserConfig represents the subset of .goreleaser.yaml we need.
type goreleaserConfig struct {
	Archives []struct {
		NameTemplate string   `yaml:"name_template"`
		Formats      []string `yaml:"formats"`
	} `yaml:"archives"`
}

// extractGoreleaserFilename parses .goreleaser.yaml and returns the asset
// filename for a given version, os, and arch.
func extractGoreleaserFilename(t *testing.T, root, version, goos, arch string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("cannot read .goreleaser.yaml: %v", err)
	}

	var cfg goreleaserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("cannot parse .goreleaser.yaml: %v", err)
	}

	if len(cfg.Archives) == 0 {
		t.Fatal(".goreleaser.yaml has no archives section")
	}

	archive := cfg.Archives[0]
	if len(archive.Formats) == 0 {
		t.Fatal(".goreleaser.yaml archive has no formats")
	}

	tmpl := archive.NameTemplate
	ext := archive.Formats[0]

	// goreleaser templates use {{ .Version }} (without v prefix),
	// {{ .Os }}, {{ .Arch }}
	name := tmpl
	name = strings.ReplaceAll(name, "{{ .Version }}", version)
	name = strings.ReplaceAll(name, "{{.Version}}", version)
	name = strings.ReplaceAll(name, "{{ .Os }}", goos)
	name = strings.ReplaceAll(name, "{{.Os}}", goos)
	name = strings.ReplaceAll(name, "{{ .Arch }}", arch)
	name = strings.ReplaceAll(name, "{{.Arch}}", arch)

	return name + "." + ext
}

// extractInstallScriptFilename parses install.sh's construct_url function and
// returns the asset filename for a given version, os, and arch.
func extractInstallScriptFilename(t *testing.T, root, version, goos, arch string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, "scripts", "install.sh"))
	if err != nil {
		t.Fatalf("cannot read install.sh: %v", err)
	}

	content := string(data)

	// Find the construct_url function and extract the filename pattern from the
	// echo statement. The line looks like:
	//   echo "https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${version_no_v}_${os}_${arch}.tar.gz"
	re := regexp.MustCompile(`(?s)construct_url\(\)\s*\{.*?echo\s+"[^"]*/([\$\{A-Za-z_\}]+\.tar\.gz)"`)
	matches := re.FindStringSubmatch(content)
	if matches == nil {
		t.Fatal("cannot find construct_url filename pattern in install.sh")
	}

	pattern := matches[1]

	// The script uses BINARY_NAME="tick" and version_no_v="${version#v}"
	// Extract BINARY_NAME value from the script.
	binaryRe := regexp.MustCompile(`BINARY_NAME="([^"]+)"`)
	binaryMatches := binaryRe.FindStringSubmatch(content)
	if binaryMatches == nil {
		t.Fatal("cannot find BINARY_NAME in install.sh")
	}
	binaryName := binaryMatches[1]

	// Substitute variables.
	result := pattern
	result = strings.ReplaceAll(result, "${BINARY_NAME}", binaryName)
	result = strings.ReplaceAll(result, "${version_no_v}", version)
	result = strings.ReplaceAll(result, "${os}", goos)
	result = strings.ReplaceAll(result, "${arch}", arch)

	return result
}

// extractHomebrewFilename parses the Homebrew formula and returns the asset
// filename for a given version, os, and arch.
func extractHomebrewFilename(t *testing.T, root, version, goos, arch string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(root, "homebrew-tap", "Formula", "tick.rb"))
	if err != nil {
		t.Fatalf("cannot read tick.rb: %v", err)
	}

	content := string(data)

	// Extract all URL patterns. The formula uses patterns like:
	//   url "https://github.com/.../tick_#{version}_darwin_arm64.tar.gz"
	// We need to find the one matching our os/arch.
	urlRe := regexp.MustCompile(`url\s+"[^"]*/([\w#\{\}_]+\.tar\.gz)"`)
	allMatches := urlRe.FindAllStringSubmatch(content, -1)
	if len(allMatches) == 0 {
		t.Fatal("cannot find URL patterns in tick.rb")
	}

	// Look for a URL that matches our os and arch.
	target := fmt.Sprintf("_%s_%s.tar.gz", goos, arch)
	for _, m := range allMatches {
		if strings.HasSuffix(m[1], target) {
			// Substitute #{version} with our version.
			filename := m[1]
			filename = strings.ReplaceAll(filename, "#{version}", version)
			return filename
		}
	}

	t.Fatalf("no Homebrew URL found for %s/%s", goos, arch)
	return ""
}

func TestAssetNamingContract(t *testing.T) {
	root := testutil.FindRepoRoot(t)

	tests := []struct {
		name    string
		version string
		goos    string
		arch    string
		want    string
	}{
		{"darwin arm64", "1.2.3", "darwin", "arm64", "tick_1.2.3_darwin_arm64.tar.gz"},
		{"darwin amd64", "1.2.3", "darwin", "amd64", "tick_1.2.3_darwin_amd64.tar.gz"},
		{"linux amd64", "0.5.0", "linux", "amd64", "tick_0.5.0_linux_amd64.tar.gz"},
		{"linux arm64", "10.20.30", "linux", "arm64", "tick_10.20.30_linux_arm64.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goreleaser := extractGoreleaserFilename(t, root, tt.version, tt.goos, tt.arch)
			installScript := extractInstallScriptFilename(t, root, tt.version, tt.goos, tt.arch)

			if goreleaser != tt.want {
				t.Errorf("goreleaser produces %q, want %q", goreleaser, tt.want)
			}
			if installScript != tt.want {
				t.Errorf("install script produces %q, want %q", installScript, tt.want)
			}
			if goreleaser != installScript {
				t.Errorf("goreleaser (%q) and install script (%q) disagree on filename", goreleaser, installScript)
			}

			// Homebrew only has darwin URLs, so only check for darwin tuples.
			if tt.goos == "darwin" {
				homebrew := extractHomebrewFilename(t, root, tt.version, tt.goos, tt.arch)
				if homebrew != tt.want {
					t.Errorf("homebrew produces %q, want %q", homebrew, tt.want)
				}
				if goreleaser != homebrew {
					t.Errorf("goreleaser (%q) and homebrew (%q) disagree on filename", goreleaser, homebrew)
				}
			}
		})
	}
}
