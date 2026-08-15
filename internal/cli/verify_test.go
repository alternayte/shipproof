package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyWithChangeIDDelegatesToRun(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	if err := os.WriteFile(filepath.Join(root, ".shipproof", "config.yaml"),
		[]byte("verification:\n  command: echo ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestChangeRecord(t, root, "SP-016")

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"verify", "SP-016"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "passed") {
		t.Errorf("expected passed in output, got: %s", stdout.String())
	}

	runJSON := filepath.Join(root, ".shipproof", "runs", "SP-016", "run.json")
	if _, err := os.Stat(runJSON); err != nil {
		t.Errorf("expected run.json written: %v", err)
	}
}

func TestVerifyAdhoc(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	if err := os.WriteFile(filepath.Join(root, ".shipproof", "config.yaml"),
		[]byte("verification:\n  command: echo adhoc-ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"verify"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "passed") {
		t.Errorf("expected passed in output, got: %s", stdout.String())
	}

	logPath := filepath.Join(root, ".shipproof", "runs", "adhoc", "stdout.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("expected adhoc stdout log: %v", err)
	}
}

func TestVerifyAdhocReturnsCommandExitCode(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	if err := os.WriteFile(filepath.Join(root, ".shipproof", "config.yaml"),
		[]byte("verification:\n  command: exit 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"verify"}, stdout, stderr)
	if code != 7 {
		t.Fatalf("expected exit 7, got %d", code)
	}
}

func TestVerifyTooManyArgs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"verify", "a", "b"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestVerifyMissingConfig(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"verify"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
}
