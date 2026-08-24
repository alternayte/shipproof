package change

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStalenessCurrent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "change-src.md")
	content := "# Source\n"
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Start(root, "SP-100", source, "", DefaultCeremony)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	stale, err := record.Staleness(root)
	if err != nil {
		t.Fatalf("Staleness() error = %v", err)
	}
	if stale.Stale {
		t.Error("expected current intent")
	}
	if stale.CurrentHash != record.SHA256 {
		t.Error("current hash must match the recorded hash")
	}
}

func TestStalenessAfterSourceChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "change-src.md")
	if err := os.WriteFile(source, []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Start(root, "SP-100", source, "", DefaultCeremony)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := os.WriteFile(source, []byte("# Source\n\nChanged intent.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, err := record.Staleness(root)
	if err != nil {
		t.Fatalf("Staleness() error = %v", err)
	}
	if !stale.Stale {
		t.Error("expected stale intent after source change")
	}
	if stale.CurrentHash == record.SHA256 {
		t.Error("current hash must differ from snapshot hash")
	}
	if stale.SnapshotHash != record.SHA256 {
		t.Error("snapshot hash must equal the recorded hash")
	}
}

func TestStalenessAfterSourceRemoval(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "change-src.md")
	if err := os.WriteFile(source, []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Start(root, "SP-100", source, "", DefaultCeremony)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}

	stale, err := record.Staleness(root)
	if err != nil {
		t.Fatalf("Staleness() error = %v", err)
	}
	if !stale.Stale {
		t.Error("expected stale intent when the source is missing")
	}
	if stale.CurrentHash != "" {
		t.Errorf("expected empty current hash, got %q", stale.CurrentHash)
	}
}
