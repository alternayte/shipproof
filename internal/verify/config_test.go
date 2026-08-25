package verify

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configContent := "version: 1\nverification:\n  command: just verify\n"
	if err := os.WriteFile(filepath.Join(shipproofDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Command != "just verify" {
		t.Fatalf("command = %q, want %q", cfg.Command, "just verify")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := LoadConfig(root)
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadConfigMissingKey(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configContent := "version: 1\n"
	if err := os.WriteFile(filepath.Join(shipproofDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(root)
	if err == nil {
		t.Fatal("expected error for missing verification command")
	}
}

func TestLoadConfigEmptyCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configContent := "version: 1\nverification:\n  command:\n"
	if err := os.WriteFile(filepath.Join(shipproofDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(root)
	if !errors.Is(err, ErrCommandMissing) {
		t.Fatalf("expected ErrCommandMissing, got %v", err)
	}
}

func TestLoadConfigCommandWithQuotes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configContent := `version: 1
verification:
  command: "go test ./..."
`
	if err := os.WriteFile(filepath.Join(shipproofDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Command != "go test ./..." {
		t.Fatalf("command = %q, want %q", cfg.Command, "go test ./...")
	}
}

func TestLoadConfigIgnoresNestedCoverageCommand(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `verification:
  coverage:
    command: go test -coverprofile={{profile}} {{target}}
  command: just verify
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Command != "just verify" {
		t.Errorf("command = %q, want just verify", cfg.Command)
	}
}
