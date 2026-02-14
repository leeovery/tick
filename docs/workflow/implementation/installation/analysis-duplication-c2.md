AGENT: duplication
FINDINGS:
- FINDING: Repeated install.sh content reading in install_test.go
  SEVERITY: medium
  FILES: scripts/install_test.go:58-66, scripts/install_test.go:69-75, scripts/install_test.go:396-401, scripts/install_test.go:866-870, scripts/install_test.go:879-885, scripts/install_test.go:893-898, scripts/install_test.go:913-918, scripts/install_test.go:1089-1094
  DESCRIPTION: Eight tests independently read install.sh contents using the same 4-line sequence (scriptPath -> os.ReadFile -> error check -> string(data)). The pattern appears in TestInstallScript (shebang, set -euo checks), TestTrapCleanup, and TestErrorHandlingHardening (main function, curl flags, interactive commands). Each block is identical: path := scriptPath(t); data, err := os.ReadFile(path); if err != nil { t.Fatalf("cannot read install.sh: %v", err) }; content := string(data). This is a clear Rule of Three violation at 8 occurrences.
  RECOMMENDATION: Extract a loadScript(t *testing.T) string helper within install_test.go (following the same pattern as loadFormula and loadWorkflow in their respective test files). All 8 call sites reduce to a single line: content := loadScript(t).

- FINDING: Duplicated test install directory setup across TestFullInstallFlow and TestErrorHandlingHardening
  SEVERITY: low
  FILES: scripts/install_test.go:512-517, scripts/install_test.go:545-549, scripts/install_test.go:583-587, scripts/install_test.go:623-627, scripts/install_test.go:935-938, scripts/install_test.go:963-967, scripts/install_test.go:995-998
  DESCRIPTION: Seven tests repeat the same 4-line block to create a temporary install directory (t.TempDir, filepath.Join for bin, os.MkdirAll, error check). The block is short and idiomatic Go test setup, but at 7 occurrences it adds noise. The six tests in TestFullInstallFlow and TestErrorHandlingHardening that run full_install mode also share a nearly identical env map structure with 6-7 common keys.
  RECOMMENDATION: Consider a makeInstallDir(t *testing.T) (tmpDir, installDir string) helper that returns both the parent temp dir and the bin subdirectory. This would cut each call site from 4 lines to 1. The env map boilerplate could optionally be reduced by a fullInstallEnv(installDir, tarball string) map[string]string builder, but this is lower priority given the per-test variations.

SUMMARY: One medium-severity finding: 8 identical file-reading blocks in install_test.go should be extracted to a loadScript helper. One low-severity finding: repeated test directory setup could be consolidated but is lower priority.
