package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build binary to temp location.
	tmpDir, err := os.MkdirTemp("", "sched-cli-test")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	binaryPath = filepath.Join(tmpDir, "sched-cli")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	// Set env to avoid touching real config.
	cmd.Env = append(os.Environ(),
		"SCHED_CONFIG_DIR="+t.TempDir(),
		"SCHED_CACHE_DIR="+t.TempDir(),
		"SCHED_LOG_DIR="+t.TempDir(),
	)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

func TestCLI_Help(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	for _, want := range []string{"sessions", "schedule", "config", "compare"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--help output missing %q", want)
		}
	}
}

func TestCLI_SessionsHelp(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "sessions", "--help")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	for _, want := range []string{"list", "show", "search"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("sessions --help output missing %q", want)
		}
	}
}

func TestCLI_ConflictingFlags(t *testing.T) {
	_, stderr, exitCode := runCLI(t, "--no-cache", "--cache-only", "sessions", "list")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for conflicting flags")
	}
	if !strings.Contains(stderr, "cannot use") {
		t.Errorf("expected stderr to contain 'cannot use', got: %s", stderr)
	}
}

func TestCLI_ConfigShow(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "config", "show")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	// Should show default config values.
	if !strings.Contains(stdout, "log_retention_days") {
		t.Errorf("config show output missing 'log_retention_days', got: %s", stdout)
	}
}

func TestCLI_RateStatus(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "rate-status")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	// Fresh store should show 0 calls.
	if !strings.Contains(stdout, "0") {
		t.Errorf("rate-status output should contain '0', got: %s", stdout)
	}
}

func TestCLI_CacheStatus(t *testing.T) {
	stdout, _, exitCode := runCLI(t, "cache", "status")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "Sessions cached") {
		t.Errorf("cache status output missing 'Sessions cached', got: %s", stdout)
	}
}
