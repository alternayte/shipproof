package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, root, content string) {
	t.Helper()
	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shipproofDir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigFull(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeConfig(t, root, `version: 1
schema_version: "0.1"
verification:
  command: just verify
evidence:
  capture: redacted
language:
  profile: ste-assisted
`)

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Verification.Command != "just verify" {
		t.Errorf("command = %q, want just verify", cfg.Verification.Command)
	}
	if cfg.Evidence.Capture != CaptureRedacted {
		t.Errorf("capture = %q, want redacted", cfg.Evidence.Capture)
	}
}

func TestLoadConfigDefaultCapture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeConfig(t, root, "verification:\n  command: just verify\n")

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Evidence.Capture != CaptureMetadata {
		t.Errorf("capture = %q, want metadata", cfg.Evidence.Capture)
	}
}

func TestLoadConfigInvalidCapture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeConfig(t, root, "verification:\n  command: just verify\nevidence:\n  capture: everything\n")

	if _, err := LoadConfig(root); err == nil {
		t.Fatal("expected error for invalid capture level")
	}
}

func TestLoadConfigMissingCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeConfig(t, root, "evidence:\n  capture: metadata\n")

	if _, err := LoadConfig(root); !errors.Is(err, ErrCommandMissing) {
		t.Fatalf("expected ErrCommandMissing, got %v", err)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := LoadConfig(root); !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestLoadEvidenceConfigMissingFileDefaultsMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg, err := LoadEvidenceConfig(root)
	if err != nil {
		t.Fatalf("LoadEvidenceConfig() error = %v", err)
	}
	if cfg.Evidence.Capture != CaptureMetadata {
		t.Errorf("capture = %q, want metadata", cfg.Evidence.Capture)
	}
}

func TestLoadEvidenceConfigWithoutCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeConfig(t, root, "evidence:\n  capture: full\n")

	cfg, err := LoadEvidenceConfig(root)
	if err != nil {
		t.Fatalf("LoadEvidenceConfig() error = %v", err)
	}
	if cfg.Evidence.Capture != CaptureFull {
		t.Errorf("capture = %q, want full", cfg.Evidence.Capture)
	}
}

func TestLoadEvidenceConfigInvalidCapture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeConfig(t, root, "evidence:\n  capture: nope\n")

	if _, err := LoadEvidenceConfig(root); err == nil {
		t.Fatal("expected error for invalid capture level")
	}
}

func TestParseConfigNestedCoverage(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `version: 1
verification:
  command: just verify
  coverage:
    command: go test -coverpkg=./... -coverprofile={{profile}} {{target}}
    format: go
  unexplained_ignore:
    - "docs/**"
    - "CHANGELOG.md"
evidence:
  capture: metadata
`)

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Verification.Command != "just verify" {
		t.Errorf("command = %q, want just verify", cfg.Verification.Command)
	}
	want := "go test -coverpkg=./... -coverprofile={{profile}} {{target}}"
	if cfg.Verification.Coverage.Command != want {
		t.Errorf("coverage command = %q, want %q", cfg.Verification.Coverage.Command, want)
	}
	if cfg.Verification.Coverage.Format != "go" {
		t.Errorf("coverage format = %q, want go", cfg.Verification.Coverage.Format)
	}
	if len(cfg.Verification.UnexplainedIgnore) != 2 ||
		cfg.Verification.UnexplainedIgnore[0] != "docs/**" ||
		cfg.Verification.UnexplainedIgnore[1] != "CHANGELOG.md" {
		t.Errorf("ignore = %#v", cfg.Verification.UnexplainedIgnore)
	}
	if cfg.Evidence.Capture != CaptureMetadata {
		t.Errorf("capture = %q", cfg.Evidence.Capture)
	}
}

func TestParseConfigWithoutCoverage(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `version: 1
verification:
  command: just verify
`)

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Verification.Coverage.Command != "" {
		t.Errorf("coverage command = %q, want empty", cfg.Verification.Coverage.Command)
	}
	if len(cfg.Verification.UnexplainedIgnore) != 0 {
		t.Errorf("ignore = %#v, want empty", cfg.Verification.UnexplainedIgnore)
	}
}

func TestParseConfigTopLevelScalarDoesNotLeakSection(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `verification:
  command: just verify
capture: full
`)

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Verification.Command != "just verify" {
		t.Errorf("command = %q, want just verify", cfg.Verification.Command)
	}
	if cfg.Evidence.Capture != CaptureMetadata {
		t.Errorf("capture = %q, want metadata", cfg.Evidence.Capture)
	}
}

func TestParseConfigTopLevelCommandDoesNotOverwriteNestedCommand(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `verification:
  command: original
command: hijack
`)

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Verification.Command != "original" {
		t.Errorf("command = %q, want original", cfg.Verification.Command)
	}
}

func TestParseConfigRealShape(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `version: 1
schema_version: "0.1"
verification:
  command: just verify
  coverage:
    command: go test -coverpkg=./... -coverprofile={{profile}} {{target}}
    format: go
  unexplained_ignore:
    - "docs/**"
    - "CHANGELOG.md"
evidence:
  capture: redacted
language:
  profile: ste-assisted
`)

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Verification.Command != "just verify" {
		t.Errorf("command = %q, want just verify", cfg.Verification.Command)
	}
	want := "go test -coverpkg=./... -coverprofile={{profile}} {{target}}"
	if cfg.Verification.Coverage.Command != want {
		t.Errorf("coverage command = %q, want %q", cfg.Verification.Coverage.Command, want)
	}
	if cfg.Verification.Coverage.Format != "go" {
		t.Errorf("coverage format = %q, want go", cfg.Verification.Coverage.Format)
	}
	if len(cfg.Verification.UnexplainedIgnore) != 2 ||
		cfg.Verification.UnexplainedIgnore[0] != "docs/**" ||
		cfg.Verification.UnexplainedIgnore[1] != "CHANGELOG.md" {
		t.Errorf("ignore = %#v", cfg.Verification.UnexplainedIgnore)
	}
	if cfg.Evidence.Capture != CaptureRedacted {
		t.Errorf("capture = %q, want redacted", cfg.Evidence.Capture)
	}
}
