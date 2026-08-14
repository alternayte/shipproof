package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeCreatesRepositoryLayout(t *testing.T) {
	root := t.TempDir()

	result, err := Initialize(root)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if len(result.CreatedDirectories) != len(directories) {
		t.Fatalf("created directories = %d, want %d", len(result.CreatedDirectories), len(directories))
	}
	if len(result.CreatedFiles) != len(initialFiles) {
		t.Fatalf("created files = %d, want %d", len(result.CreatedFiles), len(initialFiles))
	}

	for relative := range initialFiles {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", relative, err)
		}
	}
}

func TestInitializeDoesNotOverwriteExistingConfig(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("custom: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Initialize(root)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if len(result.ExistingFiles) != 1 {
		t.Fatalf("existing files = %d, want 1", len(result.ExistingFiles))
	}

	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "custom: true\n" {
		t.Fatal("existing config was overwritten")
	}
}
