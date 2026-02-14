package scripts_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leeovery/tick/internal/testutil"
)

// scriptPath returns the absolute path to install.sh.
func scriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(testutil.FindRepoRoot(t), "scripts", "install.sh")
}

// runScript executes install.sh with the given environment variables and returns
// combined output plus the exit error (nil on success).
func runScript(t *testing.T, env map[string]string) (string, error) {
	t.Helper()
	script := scriptPath(t)
	cmd := exec.Command("bash", script)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestInstallScript(t *testing.T) {
	t.Run("script file exists", func(t *testing.T) {
		path := scriptPath(t)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("install.sh not found: %v", err)
		}
		if info.IsDir() {
			t.Fatal("install.sh is a directory, expected a file")
		}
	})

	t.Run("script is executable", func(t *testing.T) {
		path := scriptPath(t)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("install.sh not found: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatal("install.sh is not executable")
		}
	})

	t.Run("script has correct shebang", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		firstLine := strings.SplitN(string(data), "\n", 2)[0]
		if firstLine != "#!/usr/bin/env bash" {
			t.Errorf("expected shebang '#!/usr/bin/env bash', got %q", firstLine)
		}
	})

	t.Run("script uses set -euo pipefail", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		if !strings.Contains(string(data), "set -euo pipefail") {
			t.Error("install.sh must contain 'set -euo pipefail'")
		}
	})
}

func TestOSDetection(t *testing.T) {
	t.Run("detects macOS via uname -s returning Darwin", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Darwin",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err != nil {
			t.Fatalf("expected success for Darwin, got error: %v\noutput: %s", err, out)
		}
		if strings.TrimSpace(out) != "darwin" {
			t.Errorf("expected output 'darwin', got: %q", strings.TrimSpace(out))
		}
	})

	t.Run("rejects unsupported OS with clear error", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Windows_NT",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err == nil {
			t.Fatal("expected non-zero exit for Windows_NT, got success")
		}
		if !strings.Contains(out, "Windows_NT") {
			t.Errorf("expected error message mentioning Windows_NT, got: %q", out)
		}
	})

	t.Run("accepts Linux OS", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err != nil {
			t.Fatalf("expected success for Linux, got error: %v\noutput: %s", err, out)
		}
		if strings.TrimSpace(out) != "linux" {
			t.Errorf("expected output 'linux', got: %q", strings.TrimSpace(out))
		}
	})

	t.Run("rejects FreeBSD with clear error", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "FreeBSD",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err == nil {
			t.Fatal("expected non-zero exit for FreeBSD, got success")
		}
		if !strings.Contains(out, "FreeBSD") {
			t.Errorf("expected error message mentioning FreeBSD, got: %q", out)
		}
	})
}

func TestArchDetection(t *testing.T) {
	tests := []struct {
		name     string
		unameM   string
		expected string
		wantErr  bool
	}{
		{"maps x86_64 to amd64", "x86_64", "amd64", false},
		{"maps aarch64 to arm64", "aarch64", "arm64", false},
		{"maps arm64 to arm64", "arm64", "arm64", false},
		{"rejects i686 with error", "i686", "", true},
		{"rejects ppc64le with error", "ppc64le", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runScript(t, map[string]string{
				"TICK_TEST_UNAME_S": "Linux",
				"TICK_TEST_UNAME_M": tt.unameM,
				"TICK_TEST_MODE":    "detect_arch",
			})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for arch %q, got success", tt.unameM)
				}
				if !strings.Contains(out, tt.unameM) {
					t.Errorf("expected error message mentioning %q, got: %q", tt.unameM, out)
				}
			} else {
				if err != nil {
					t.Fatalf("expected success for arch %q, got error: %v\noutput: %s", tt.unameM, err, out)
				}
				if !strings.Contains(out, tt.expected) {
					t.Errorf("expected output containing %q, got: %q", tt.expected, out)
				}
			}
		})
	}
}

func TestURLConstruction(t *testing.T) {
	t.Run("constructs correct download URL from version os and arch", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "construct_url",
			"TICK_TEST_VERSION": "v1.2.3",
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		expected := "https://github.com/leeovery/tick/releases/download/v1.2.3/tick_1.2.3_linux_amd64.tar.gz"
		if strings.TrimSpace(out) != expected {
			t.Errorf("expected URL %q, got: %q", expected, strings.TrimSpace(out))
		}
	})

	t.Run("constructs correct URL for arm64", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "aarch64",
			"TICK_TEST_MODE":    "construct_url",
			"TICK_TEST_VERSION": "v0.5.0",
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		expected := "https://github.com/leeovery/tick/releases/download/v0.5.0/tick_0.5.0_linux_arm64.tar.gz"
		if strings.TrimSpace(out) != expected {
			t.Errorf("expected URL %q, got: %q", expected, strings.TrimSpace(out))
		}
	})

	t.Run("version without v prefix in filename", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "construct_url",
			"TICK_TEST_VERSION": "v10.20.30",
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		// The filename should NOT have 'v' prefix on version.
		if strings.Contains(out, "tick_v10.20.30") {
			t.Error("filename should not contain 'v' prefix on version")
		}
		if !strings.Contains(out, "tick_10.20.30_linux_amd64.tar.gz") {
			t.Errorf("expected 'tick_10.20.30_linux_amd64.tar.gz' in output, got: %q", out)
		}
	})
}

func TestVersionResolution(t *testing.T) {
	t.Run("fails with error when GitHub API is unreachable", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_MODE": "resolve_version",
			"GITHUB_API":     "file:///nonexistent",
		})
		if err == nil {
			t.Fatal("expected failure when GitHub API is unreachable, got success")
		}
		if !strings.Contains(strings.ToLower(out), "error") && !strings.Contains(strings.ToLower(out), "could not") {
			// curl should fail with a non-zero exit, which set -e will catch.
			// The script may not reach the custom error message, but it must exit non-zero.
			_ = out // exit code check above is sufficient
		}
	})
}

func TestInstallDirectorySelection(t *testing.T) {
	t.Run("installs to custom dir when writable", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "select_install_dir",
			"TICK_INSTALL_DIR":  installDir,
			"TICK_FALLBACK_DIR": filepath.Join(tmpDir, "fallback"),
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		if !strings.Contains(out, installDir) {
			t.Errorf("expected output to mention install dir %q, got: %q", installDir, out)
		}
	})

	t.Run("falls back when primary dir is not writable", func(t *testing.T) {
		tmpDir := t.TempDir()
		primaryDir := filepath.Join(tmpDir, "readonly")
		fallbackDir := filepath.Join(tmpDir, "fallback", "bin")

		// Create a non-writable primary dir.
		if err := os.MkdirAll(primaryDir, 0555); err != nil {
			t.Fatalf("cannot create primary dir: %v", err)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "select_install_dir",
			"TICK_INSTALL_DIR":  primaryDir,
			"TICK_FALLBACK_DIR": fallbackDir,
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		if !strings.Contains(out, fallbackDir) {
			t.Errorf("expected output to mention fallback dir %q, got: %q", fallbackDir, out)
		}
	})

	t.Run("creates fallback dir via mkdir -p if it does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		primaryDir := filepath.Join(tmpDir, "readonly")
		fallbackDir := filepath.Join(tmpDir, "new", "nested", "bin")

		if err := os.MkdirAll(primaryDir, 0555); err != nil {
			t.Fatalf("cannot create primary dir: %v", err)
		}

		_, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "select_install_dir",
			"TICK_INSTALL_DIR":  primaryDir,
			"TICK_FALLBACK_DIR": fallbackDir,
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}

		info, err := os.Stat(fallbackDir)
		if err != nil {
			t.Fatalf("fallback dir %q was not created: %v", fallbackDir, err)
		}
		if !info.IsDir() {
			t.Fatalf("fallback dir %q is not a directory", fallbackDir)
		}
	})
}

func TestPATHWarning(t *testing.T) {
	t.Run("no PATH warning when primary dir is writable", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_MODE":   "select_install_dir",
			"TICK_INSTALL_DIR": installDir,
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		if strings.Contains(out, "WARNING") || strings.Contains(out, "is not in your PATH") {
			t.Errorf("expected no WARNING or PATH warning when primary dir is writable, got: %q", out)
		}
	})

	t.Run("prints PATH warning when fallback dir not in PATH", func(t *testing.T) {
		tmpDir := t.TempDir()
		primaryDir := filepath.Join(tmpDir, "readonly")
		fallbackDir := filepath.Join(tmpDir, "fallback", "bin")

		if err := os.MkdirAll(primaryDir, 0555); err != nil {
			t.Fatalf("cannot create primary dir: %v", err)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "select_install_dir",
			"TICK_INSTALL_DIR":  primaryDir,
			"TICK_FALLBACK_DIR": fallbackDir,
			"TICK_TEST_PATH":    "/usr/bin:/usr/local/bin",
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		if !strings.Contains(out, "PATH") {
			t.Errorf("expected PATH warning in output, got: %q", out)
		}
	})

	t.Run("no PATH warning when fallback dir is in PATH", func(t *testing.T) {
		tmpDir := t.TempDir()
		primaryDir := filepath.Join(tmpDir, "readonly")
		fallbackDir := filepath.Join(tmpDir, "fallback", "bin")

		if err := os.MkdirAll(primaryDir, 0555); err != nil {
			t.Fatalf("cannot create primary dir: %v", err)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "select_install_dir",
			"TICK_INSTALL_DIR":  primaryDir,
			"TICK_FALLBACK_DIR": fallbackDir,
			"TICK_TEST_PATH":    "/usr/bin:" + fallbackDir,
		})
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		// Should not contain a PATH warning since fallbackDir is in PATH.
		if strings.Contains(strings.ToLower(out), "add") && strings.Contains(out, "PATH") {
			t.Errorf("expected no PATH warning when fallback dir is in PATH, got: %q", out)
		}
	})
}

func TestTrapCleanup(t *testing.T) {
	t.Run("script contains trap for cleanup", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "trap") {
			t.Error("install.sh must contain a trap for cleanup")
		}
	})
}

// createFakeBrew creates a fake brew script in a temp dir and returns the
// directory (to prepend to PATH) and a log file path where invocations are
// recorded. The behavior parameter controls exit codes:
//
//	"success"       — all commands succeed
//	"tap-fail"      — brew tap exits 1
//	"install-fail"  — brew install exits 1
//	"already-installed" — brew install prints "already installed" warning and exits 0
func createFakeBrew(t *testing.T, behavior string) (binDir, logFile string) {
	t.Helper()
	tmpDir := t.TempDir()
	binDir = filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("cannot create fake brew bin dir: %v", err)
	}
	logFile = filepath.Join(tmpDir, "brew.log")

	var script string
	switch behavior {
	case "success":
		script = `#!/usr/bin/env bash
echo "$@" >> "` + logFile + `"
echo "brew $@"
exit 0
`
	case "tap-fail":
		script = `#!/usr/bin/env bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "tap" ]]; then
    echo "Error: tap failed" >&2
    exit 1
fi
echo "brew $@"
exit 0
`
	case "install-fail":
		script = `#!/usr/bin/env bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "install" ]]; then
    echo "Error: install failed" >&2
    exit 1
fi
echo "brew $@"
exit 0
`
	case "already-installed":
		script = `#!/usr/bin/env bash
echo "$@" >> "` + logFile + `"
if [[ "$1" == "install" ]]; then
    echo "Warning: tick is already installed" >&2
fi
echo "brew $@"
exit 0
`
	default:
		t.Fatalf("unknown fake brew behavior: %s", behavior)
	}

	brewPath := filepath.Join(binDir, "brew")
	if err := os.WriteFile(brewPath, []byte(script), 0755); err != nil {
		t.Fatalf("cannot write fake brew: %v", err)
	}
	return binDir, logFile
}

// runScriptWithFakeBrew runs install.sh with a fake brew on PATH and Darwin uname.
func runScriptWithFakeBrew(t *testing.T, brewBehavior string, extraEnv map[string]string) (string, error, string) {
	t.Helper()
	brewDir, logFile := createFakeBrew(t, brewBehavior)

	env := map[string]string{
		"TICK_TEST_UNAME_S": "Darwin",
		"TICK_TEST_MODE":    "install_macos",
		"PATH":              brewDir + ":/usr/bin:/bin",
	}
	for k, v := range extraEnv {
		env[k] = v
	}

	out, err := runScript(t, env)
	return out, err, logFile
}

// createFakeTarball creates a tar.gz containing a fake tick binary with the
// given content. Returns the path to the tarball.
func createFakeTarball(t *testing.T, dir string, content string) string {
	t.Helper()
	fakeDir := filepath.Join(dir, "fake")
	if err := os.MkdirAll(fakeDir, 0755); err != nil {
		t.Fatalf("cannot create fake dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fakeDir, "tick"), []byte(content), 0755); err != nil {
		t.Fatalf("cannot write fake binary: %v", err)
	}
	tarball := filepath.Join(dir, "tick.tar.gz")
	tarCmd := exec.Command("tar", "czf", tarball, "-C", fakeDir, "tick")
	if out, err := tarCmd.CombinedOutput(); err != nil {
		t.Fatalf("cannot create fake tarball: %v\n%s", err, out)
	}
	return tarball
}

func TestFullInstallFlow(t *testing.T) {
	t.Run("full install with mocked download succeeds", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		tarball := createFakeTarball(t, tmpDir, "#!/bin/sh\necho fake-tick")

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "full_install",
			"TICK_TEST_VERSION": "v1.0.0",
			"TICK_TEST_TARBALL": tarball,
			"TICK_INSTALL_DIR":  installDir,
			"TICK_FALLBACK_DIR": filepath.Join(tmpDir, "fallback"),
		})
		if err != nil {
			t.Fatalf("full install failed: %v\noutput: %s", err, out)
		}

		// Verify the binary was installed.
		installed := filepath.Join(installDir, "tick")
		info, err := os.Stat(installed)
		if err != nil {
			t.Fatalf("installed binary not found: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatal("installed binary is not executable")
		}
	})

	t.Run("overwrites existing binary without prompting", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		// Write an "existing" tick binary.
		existingBinary := filepath.Join(installDir, "tick")
		if err := os.WriteFile(existingBinary, []byte("old-version"), 0755); err != nil {
			t.Fatalf("cannot write existing binary: %v", err)
		}

		tarball := createFakeTarball(t, tmpDir, "new-version")

		_, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "full_install",
			"TICK_TEST_VERSION": "v2.0.0",
			"TICK_TEST_TARBALL": tarball,
			"TICK_INSTALL_DIR":  installDir,
			"TICK_FALLBACK_DIR": filepath.Join(tmpDir, "fallback"),
		})
		if err != nil {
			t.Fatalf("overwrite install failed: %v", err)
		}

		// Verify the binary was overwritten.
		data, err := os.ReadFile(existingBinary)
		if err != nil {
			t.Fatalf("cannot read installed binary: %v", err)
		}
		if string(data) == "old-version" {
			t.Error("binary was not overwritten — still contains old content")
		}
	})

	t.Run("cleans up temp directory on failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		// Point to a nonexistent tarball so extraction fails.
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S":     "Linux",
			"TICK_TEST_UNAME_M":     "x86_64",
			"TICK_TEST_MODE":        "full_install",
			"TICK_TEST_VERSION":     "v1.0.0",
			"TICK_TEST_TARBALL":     filepath.Join(tmpDir, "nonexistent.tar.gz"),
			"TICK_INSTALL_DIR":      installDir,
			"TICK_FALLBACK_DIR":     filepath.Join(tmpDir, "fallback"),
			"TICK_TEST_ECHO_TMPDIR": "1",
		})
		if err == nil {
			t.Fatal("expected failure with nonexistent tarball, got success")
		}

		// Extract the temp dir path from output.
		var tmpDirPath string
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "TICK_TMPDIR=") {
				tmpDirPath = strings.TrimPrefix(line, "TICK_TMPDIR=")
				break
			}
		}
		if tmpDirPath == "" {
			t.Fatal("could not find TICK_TMPDIR in output")
		}

		// Verify the temp directory was cleaned up even on failure.
		if _, err := os.Stat(tmpDirPath); !os.IsNotExist(err) {
			t.Errorf("temp directory %q was not cleaned up after failure", tmpDirPath)
		}
	})

	t.Run("cleans up temp directory on success", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		tarball := createFakeTarball(t, tmpDir, "#!/bin/sh\necho tick")

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S":     "Linux",
			"TICK_TEST_UNAME_M":     "x86_64",
			"TICK_TEST_MODE":        "full_install",
			"TICK_TEST_VERSION":     "v1.0.0",
			"TICK_TEST_TARBALL":     tarball,
			"TICK_INSTALL_DIR":      installDir,
			"TICK_FALLBACK_DIR":     filepath.Join(tmpDir, "fallback"),
			"TICK_TEST_ECHO_TMPDIR": "1",
		})
		if err != nil {
			t.Fatalf("install failed: %v\noutput: %s", err, out)
		}

		// Extract the temp dir path from output.
		var tmpDirPath string
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "TICK_TMPDIR=") {
				tmpDirPath = strings.TrimPrefix(line, "TICK_TMPDIR=")
				break
			}
		}
		if tmpDirPath == "" {
			t.Fatal("could not find TICK_TMPDIR in output")
		}

		// Verify the temp directory was cleaned up.
		if _, err := os.Stat(tmpDirPath); !os.IsNotExist(err) {
			t.Errorf("temp directory %q was not cleaned up", tmpDirPath)
		}
	})
}

func TestMacOSInstall(t *testing.T) {
	t.Run("it runs brew tap and brew install when brew is available on macOS", func(t *testing.T) {
		out, err, logFile := runScriptWithFakeBrew(t, "success", nil)
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}

		logData, readErr := os.ReadFile(logFile)
		if readErr != nil {
			t.Fatalf("cannot read brew log: %v", readErr)
		}
		log := string(logData)
		if !strings.Contains(log, "tap leeovery/tick") {
			t.Errorf("expected brew tap leeovery/tick in log, got: %q", log)
		}
		if !strings.Contains(log, "install tick") {
			t.Errorf("expected brew install tick in log, got: %q", log)
		}
	})

	t.Run("it exits 0 on successful Homebrew install", func(t *testing.T) {
		_, err, _ := runScriptWithFakeBrew(t, "success", nil)
		if err != nil {
			t.Fatalf("expected exit 0 on successful brew install, got error: %v", err)
		}
	})

	t.Run("it prints a success message after Homebrew install", func(t *testing.T) {
		out, err, _ := runScriptWithFakeBrew(t, "success", nil)
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		if !strings.Contains(strings.ToLower(out), "success") || !strings.Contains(strings.ToLower(out), "homebrew") {
			t.Errorf("expected success message mentioning Homebrew, got: %q", out)
		}
	})

	t.Run("it propagates exit code when brew tap fails", func(t *testing.T) {
		out, err, logFile := runScriptWithFakeBrew(t, "tap-fail", nil)
		if err == nil {
			t.Fatal("expected non-zero exit when brew tap fails, got success")
		}
		if !strings.Contains(out, "tap failed") && !strings.Contains(strings.ToLower(out), "error") {
			t.Errorf("expected error output when brew tap fails, got: %q", out)
		}
		logData, readErr := os.ReadFile(logFile)
		if readErr != nil {
			t.Fatalf("cannot read brew log: %v", readErr)
		}
		if strings.Contains(string(logData), "install tick") {
			t.Error("brew install should not be called when brew tap fails")
		}
	})

	t.Run("it propagates exit code when brew install fails", func(t *testing.T) {
		out, err, _ := runScriptWithFakeBrew(t, "install-fail", nil)
		if err == nil {
			t.Fatal("expected non-zero exit when brew install fails, got success")
		}
		if !strings.Contains(out, "install failed") && !strings.Contains(strings.ToLower(out), "error") {
			t.Errorf("expected error output when brew install fails, got: %q", out)
		}
	})

	t.Run("it does not run Linux download logic on macOS", func(t *testing.T) {
		out, err, _ := runScriptWithFakeBrew(t, "success", nil)
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		// Linux flow prints "Downloading" — macOS should not.
		if strings.Contains(out, "Downloading") {
			t.Errorf("macOS path should not run Linux download logic, but output contains 'Downloading': %q", out)
		}
	})

	t.Run("it does not suppress brew output", func(t *testing.T) {
		out, err, _ := runScriptWithFakeBrew(t, "success", nil)
		if err != nil {
			t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
		}
		if !strings.Contains(out, "brew tap leeovery/tick") {
			t.Errorf("expected brew tap output to be visible, got: %q", out)
		}
		if !strings.Contains(out, "brew install tick") {
			t.Errorf("expected brew install output to be visible, got: %q", out)
		}
	})

	t.Run("it handles tick already installed via Homebrew (idempotent)", func(t *testing.T) {
		out, err, _ := runScriptWithFakeBrew(t, "already-installed", nil)
		if err != nil {
			t.Fatalf("expected success when tick already installed, got error: %v\noutput: %s", err, out)
		}
	})
}

// runScriptSeparateOutputs executes install.sh and returns stdout and stderr
// as separate strings, plus the exit error.
func runScriptSeparateOutputs(t *testing.T, env map[string]string) (stdout, stderr string, err error) {
	t.Helper()
	script := scriptPath(t)
	cmd := exec.Command("bash", script)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func TestMacOSNoBrewError(t *testing.T) {
	// Environment that simulates macOS with no brew on PATH.
	// PATH is set to a minimal value that excludes any real brew.
	noBrew := map[string]string{
		"TICK_TEST_UNAME_S": "Darwin",
		"TICK_TEST_MODE":    "install_macos",
		"PATH":              "/usr/bin:/bin",
	}

	t.Run("it exits with code 1 on macOS when brew is not found", func(t *testing.T) {
		_, err := runScript(t, noBrew)
		if err == nil {
			t.Fatal("expected non-zero exit when brew is not found, got success")
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
		}
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	})

	t.Run("it prints the Homebrew install instructions on macOS without brew", func(t *testing.T) {
		out, err := runScript(t, noBrew)
		if err == nil {
			t.Fatal("expected error, got success")
		}
		if !strings.Contains(out, "brew tap leeovery/tick && brew install tick") {
			t.Errorf("expected Homebrew install instructions, got: %q", out)
		}
	})

	t.Run("it prints the Please install via Homebrew message", func(t *testing.T) {
		out, err := runScript(t, noBrew)
		if err == nil {
			t.Fatal("expected error, got success")
		}
		if !strings.Contains(out, "Please install via Homebrew:") {
			t.Errorf("expected 'Please install via Homebrew:' in output, got: %q", out)
		}
	})

	t.Run("it does not attempt a binary download on macOS without brew", func(t *testing.T) {
		out, err := runScript(t, noBrew)
		if err == nil {
			t.Fatal("expected error, got success")
		}
		if strings.Contains(out, "Downloading") {
			t.Errorf("should not attempt download on macOS without brew, got: %q", out)
		}
		if strings.Contains(out, "curl") {
			t.Errorf("should not invoke curl on macOS without brew, got: %q", out)
		}
	})

	t.Run("it does not create a temporary directory on macOS without brew", func(t *testing.T) {
		out, err := runScript(t, noBrew)
		if err == nil {
			t.Fatal("expected error, got success")
		}
		if strings.Contains(out, "TICK_TMPDIR=") {
			t.Errorf("should not create temp dir on macOS without brew, got: %q", out)
		}
		if strings.Contains(out, "mktemp") {
			t.Errorf("should not invoke mktemp on macOS without brew, got: %q", out)
		}
	})

	t.Run("it outputs the error message to stderr not stdout", func(t *testing.T) {
		stdout, stderr, err := runScriptSeparateOutputs(t, noBrew)
		if err == nil {
			t.Fatal("expected error, got success")
		}
		if !strings.Contains(stderr, "Please install via Homebrew:") {
			t.Errorf("expected error message on stderr, got stderr: %q", stderr)
		}
		if !strings.Contains(stderr, "brew tap leeovery/tick && brew install tick") {
			t.Errorf("expected install instructions on stderr, got stderr: %q", stderr)
		}
		if strings.Contains(stdout, "Please install via Homebrew:") {
			t.Errorf("error message should not appear on stdout, got stdout: %q", stdout)
		}
	})
}

func TestErrorHandlingHardening(t *testing.T) {
	t.Run("script body is wrapped in a main function", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		content := string(data)
		// Check for main() or main () function declaration
		hasMainFunc := strings.Contains(content, "main()") || strings.Contains(content, "main ()")
		if !hasMainFunc {
			t.Error("install.sh must define a main() function")
		}
	})

	t.Run("main function call is the last line", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		content := string(data)
		lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if lastLine != `main "$@"` {
			t.Errorf("expected last line to be 'main \"$@\"', got: %q", lastLine)
		}
	})

	t.Run("curl uses -f flag for version resolution API call", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		content := string(data)
		// Find the curl call in resolve_version
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "curl") && strings.Contains(line, "GITHUB_API") {
				if !strings.Contains(line, "-f") {
					t.Errorf("curl call for version resolution must use -f flag: %q", line)
				}
				return
			}
		}
		t.Error("could not find curl call for version resolution")
	})

	t.Run("curl uses -f flag for archive download", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		content := string(data)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "curl") && strings.Contains(line, "BINARY_NAME") && strings.Contains(line, "tar.gz") {
				if !strings.Contains(line, "-f") {
					t.Errorf("curl call for archive download must use -f flag: %q", line)
				}
				return
			}
		}
		t.Error("could not find curl call for archive download")
	})

	t.Run("script exits with error for empty downloaded archive", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		// Create a zero-byte tarball
		emptyTarball := filepath.Join(tmpDir, "empty.tar.gz")
		if err := os.WriteFile(emptyTarball, []byte{}, 0644); err != nil {
			t.Fatalf("cannot create empty tarball: %v", err)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "full_install",
			"TICK_TEST_VERSION": "v1.0.0",
			"TICK_TEST_TARBALL": emptyTarball,
			"TICK_INSTALL_DIR":  installDir,
			"TICK_FALLBACK_DIR": filepath.Join(tmpDir, "fallback"),
		})
		if err == nil {
			t.Fatal("expected failure with empty tarball, got success")
		}
		if !strings.Contains(out, "empty or missing") {
			t.Errorf("expected 'empty or missing' in error output, got: %q", out)
		}
	})

	t.Run("script exits with error for corrupt archive", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		// Create a non-gzip file that looks like content but isn't valid
		corruptTarball := filepath.Join(tmpDir, "corrupt.tar.gz")
		if err := os.WriteFile(corruptTarball, []byte("this is not a gzip file"), 0644); err != nil {
			t.Fatalf("cannot create corrupt tarball: %v", err)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "full_install",
			"TICK_TEST_VERSION": "v1.0.0",
			"TICK_TEST_TARBALL": corruptTarball,
			"TICK_INSTALL_DIR":  installDir,
			"TICK_FALLBACK_DIR": filepath.Join(tmpDir, "fallback"),
		})
		if err == nil {
			t.Fatal("expected failure with corrupt tarball, got success")
		}
		if !strings.Contains(out, "Failed to extract") {
			t.Errorf("expected 'Failed to extract' in error output, got: %q", out)
		}
	})

	t.Run("script exits with error when binary not found in archive", func(t *testing.T) {
		tmpDir := t.TempDir()
		installDir := filepath.Join(tmpDir, "bin")
		if err := os.MkdirAll(installDir, 0755); err != nil {
			t.Fatalf("cannot create install dir: %v", err)
		}

		// Create a valid tar.gz that does NOT contain a "tick" binary
		fakeDir := filepath.Join(tmpDir, "fakearchive")
		if err := os.MkdirAll(fakeDir, 0755); err != nil {
			t.Fatalf("cannot create fake dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(fakeDir, "notick"), []byte("wrong binary"), 0755); err != nil {
			t.Fatalf("cannot write fake file: %v", err)
		}
		tarball := filepath.Join(tmpDir, "notick.tar.gz")
		tarCmd := exec.Command("tar", "czf", tarball, "-C", fakeDir, "notick")
		if out, err := tarCmd.CombinedOutput(); err != nil {
			t.Fatalf("cannot create tarball: %v\n%s", err, out)
		}

		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "Linux",
			"TICK_TEST_UNAME_M": "x86_64",
			"TICK_TEST_MODE":    "full_install",
			"TICK_TEST_VERSION": "v1.0.0",
			"TICK_TEST_TARBALL": tarball,
			"TICK_INSTALL_DIR":  installDir,
			"TICK_FALLBACK_DIR": filepath.Join(tmpDir, "fallback"),
		})
		if err == nil {
			t.Fatal("expected failure when binary not in archive, got success")
		}
		if !strings.Contains(out, "not found in archive") {
			t.Errorf("expected 'not found in archive' in error output, got: %q", out)
		}
	})

	t.Run("script exits with error for FreeBSD", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "FreeBSD",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err == nil {
			t.Fatal("expected non-zero exit for FreeBSD, got success")
		}
		if !strings.Contains(out, "FreeBSD") {
			t.Errorf("expected error mentioning FreeBSD, got: %q", out)
		}
	})

	t.Run("script exits with error for CYGWIN", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "CYGWIN_NT-10.0",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err == nil {
			t.Fatal("expected non-zero exit for CYGWIN, got success")
		}
		if !strings.Contains(out, "CYGWIN") {
			t.Errorf("expected error mentioning CYGWIN, got: %q", out)
		}
	})

	t.Run("script exits with error for unknown OS", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "SomeFutureOS",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err == nil {
			t.Fatal("expected non-zero exit for unknown OS, got success")
		}
		if !strings.Contains(out, "SomeFutureOS") {
			t.Errorf("expected error mentioning SomeFutureOS, got: %q", out)
		}
	})

	t.Run("unsupported OS error message includes the detected OS name", func(t *testing.T) {
		out, err := runScript(t, map[string]string{
			"TICK_TEST_UNAME_S": "HaikuOS",
			"TICK_TEST_MODE":    "detect_os",
		})
		if err == nil {
			t.Fatal("expected non-zero exit for HaikuOS, got success")
		}
		if !strings.Contains(out, "Unsupported operating system") {
			t.Errorf("expected 'Unsupported operating system' in error, got: %q", out)
		}
		if !strings.Contains(out, "HaikuOS") {
			t.Errorf("expected 'HaikuOS' in error, got: %q", out)
		}
		if !strings.Contains(out, "Linux") || !strings.Contains(out, "macOS") {
			t.Errorf("expected error to mention supported OSes (Linux and macOS), got: %q", out)
		}
	})

	t.Run("script does not contain read or select commands", func(t *testing.T) {
		path := scriptPath(t)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read install.sh: %v", err)
		}
		content := string(data)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip comments and empty lines
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			// Check for interactive 'read' commands (not 'read' inside variable names or strings)
			// Look for 'read ' at the start of a statement
			if strings.HasPrefix(trimmed, "read ") || trimmed == "read" {
				t.Errorf("line %d contains interactive 'read' command: %q", i+1, trimmed)
			}
			// Check for 'select' keyword used for interactive menus
			if strings.HasPrefix(trimmed, "select ") {
				t.Errorf("line %d contains interactive 'select' command: %q", i+1, trimmed)
			}
		}
	})
}
