package change

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartAndLoad(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "test.md")
	if err := os.WriteFile(src, []byte("# Test\n\nContent."), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Start(root, "SP-001", src, "")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if record.ChangeID != "SP-001" {
		t.Fatalf("change_id = %q", record.ChangeID)
	}
	if record.SHA256 == "" {
		t.Fatal("sha256 is empty")
	}
	if record.CapturedAt == "" {
		t.Fatal("captured_at is empty")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(record.SnapshotPath))); err != nil {
		t.Fatalf("snapshot file: %v", err)
	}

	loaded, err := Load(root, "SP-001")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.SHA256 != record.SHA256 {
		t.Fatalf("sha256 mismatch: %q vs %q", loaded.SHA256, record.SHA256)
	}
}

func TestDuplicateStartFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "spec.md")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Start(root, "SP-002", src, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(root, "SP-002", src, ""); err == nil {
		t.Fatal("expected duplicate start error")
	}
}

func TestVerifyHashDetectsTampering(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "intent.md")
	if err := os.WriteFile(src, []byte("original content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Start(root, "SP-003", src, ""); err != nil {
		t.Fatal(err)
	}

	record, err := Load(root, "SP-003")
	if err != nil {
		t.Fatal(err)
	}

	if err := record.VerifyHash(root); err != nil {
		t.Fatalf("unexpected hash mismatch: %v", err)
	}

	snapshotPath := filepath.Join(root, filepath.FromSlash(record.SnapshotPath))
	if err := os.WriteFile(snapshotPath, []byte("tampered content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := record.VerifyHash(root); err == nil {
		t.Fatal("expected hash mismatch after tampering")
	}
}

func TestStartRejectsEmptyChangeID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "doc.md")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Start(root, "", src, ""); err == nil {
		t.Fatal("expected error for empty change id")
	}
}

func TestStartRejectsMissingSource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Start(root, "SP-005", filepath.Join(root, "missing.md"), ""); err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestStartWithShapingRef(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "test.md")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Start(root, "SP-020", src, "complete-metrics")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if record.ShapingRef != "complete-metrics" {
		t.Fatalf("shaping_ref = %q, want %q", record.ShapingRef, "complete-metrics")
	}

	loaded, err := Load(root, "SP-020")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ShapingRef != "complete-metrics" {
		t.Fatalf("loaded shaping_ref = %q, want %q", loaded.ShapingRef, "complete-metrics")
	}
}

func TestLoadRecordWithoutShapingRef(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "test.md")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Start(root, "SP-021", src, ""); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(root, "SP-021")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ShapingRef != "" {
		t.Fatalf("shaping_ref should be empty, got %q", loaded.ShapingRef)
	}
}
