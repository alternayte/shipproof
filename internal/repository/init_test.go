package repository

import (
	"os"
	"path/filepath"
	"strings"
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

func TestInitializeWritesALoadableConfig(t *testing.T) {
	root := t.TempDir()

	if _, err := Initialize(root); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	config, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config.Verification.Command != "just verify" {
		t.Fatalf("verification.command = %q, want %q", config.Verification.Command, "just verify")
	}
}

// TestInitWritesNoLanguagePolicy covers SP-023. The STE lint is gone, so the
// default config must hold no language policy and init must write no glossary.
func TestInitWritesNoLanguagePolicy(t *testing.T) {
	root := t.TempDir()
	if _, err := Initialize(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), "language:") {
		t.Errorf("default config still holds a language policy:\n%s", data)
	}

	if _, err := os.Stat(filepath.Join(root, ".shipproof", "glossary.yaml")); !os.IsNotExist(err) {
		t.Errorf("init still writes a glossary: %v", err)
	}
}
