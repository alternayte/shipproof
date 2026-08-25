package change

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartAndLoad(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	src := filepath.Join(root, "test.md")
	if err := os.WriteFile(src, []byte("# Test\n\nContent."), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Start(root, "SP-001", src, "", DefaultCeremony)
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

	if _, err := Start(root, "SP-002", src, "", DefaultCeremony); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(root, "SP-002", src, "", DefaultCeremony); err == nil {
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

	if _, err := Start(root, "SP-003", src, "", DefaultCeremony); err != nil {
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

	if _, err := Start(root, "", src, "", DefaultCeremony); err == nil {
		t.Fatal("expected error for empty change id")
	}
}

func TestStartRejectsMissingSource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Start(root, "SP-005", filepath.Join(root, "missing.md"), "", DefaultCeremony); err == nil {
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

	record, err := Start(root, "SP-020", src, "complete-metrics", DefaultCeremony)
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

	if _, err := Start(root, "SP-021", src, "", DefaultCeremony); err != nil {
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

func TestStartRecordsCeremonyZero(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "SP-100.md")
	if err := os.WriteFile(source, []byte("# SP-100\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Start(root, "SP-100", source, "", 0)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if record.CeremonyLevel() != 0 {
		t.Fatalf("CeremonyLevel() = %d, want 0", record.CeremonyLevel())
	}

	data, err := os.ReadFile(Path(root, "SP-100"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"ceremony": 0`) {
		t.Fatalf("record does not hold ceremony 0:\n%s", data)
	}
}

func TestLoadRecordWithoutCeremonyReadsAsOne(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, ".shipproof", "changes", "SP-001")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `{
  "schema_version": "0.1",
  "change_id": "SP-001",
  "source_path": "docs/changes/SP-001.md",
  "snapshot_path": ".shipproof/changes/SP-001/snapshot.md",
  "sha256": "511d9f04cea0429920ad7733e973fbe7741190694b4fb8bcc2b8ebbcdc478a56",
  "captured_at": "2026-08-14T21:58:11Z"
}
`
	if err := os.WriteFile(filepath.Join(dir, "change.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Load(root, "SP-001")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if record.CeremonyLevel() != 1 {
		t.Fatalf("CeremonyLevel() = %d, want 1", record.CeremonyLevel())
	}
}

func TestStartRejectsCeremonyOutOfRange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "SP-101.md")
	if err := os.WriteFile(source, []byte("# SP-101\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Start(root, "SP-101", source, "", 4); err == nil {
		t.Fatal("expected an error for ceremony 4")
	}
}

func TestRestartResnapshotsAndKeepsTheCeremonyLevel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "SP-110.md")
	if err := os.WriteFile(source, []byte("# SP-110\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := Start(root, "SP-110", source, "session-1", 2)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := os.WriteFile(source, []byte("# SP-110\n\nnew intent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	second, err := Restart(root, "SP-110", source, "", nil)
	if err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	if second.SHA256 == first.SHA256 {
		t.Fatal("Restart() did not re-snapshot the source document")
	}
	if second.CeremonyLevel() != 2 {
		t.Fatalf("CeremonyLevel() = %d, want 2", second.CeremonyLevel())
	}
	if second.ShapingRef != "session-1" {
		t.Fatalf("ShapingRef = %q, want %q", second.ShapingRef, "session-1")
	}

	staleness, err := second.Staleness(root)
	if err != nil {
		t.Fatalf("Staleness() error = %v", err)
	}
	if staleness.Stale {
		t.Fatal("the change is still stale after Restart()")
	}
}

func TestRestartAcceptsANewCeremonyLevel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "SP-111.md")
	if err := os.WriteFile(source, []byte("# SP-111\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(root, "SP-111", source, "", 2); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	level := 0
	record, err := Restart(root, "SP-111", source, "", &level)
	if err != nil {
		t.Fatalf("Restart() error = %v", err)
	}
	if record.CeremonyLevel() != 0 {
		t.Fatalf("CeremonyLevel() = %d, want 0", record.CeremonyLevel())
	}
}

func TestStartWithoutForceStillRefusesAnExistingChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "SP-112.md")
	if err := os.WriteFile(source, []byte("# SP-112\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(root, "SP-112", source, "", DefaultCeremony); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if _, err := Start(root, "SP-112", source, "", DefaultCeremony); err == nil {
		t.Fatal("expected a refusal for an existing change")
	} else if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v, want an already-exists refusal", err)
	}
}

// TestLoadV0Records proves criterion S5 of the v1 spine. Every change record
// that ShipProof v0 wrote must still load after the ceremony field arrives.
// The tracked fixtures under testdata/v0 always run. The live records under
// .shipproof/changes/ run as well when the directory exists, because the
// working repository is the strongest available sample.
func TestLoadV0Records(t *testing.T) {
	t.Parallel()

	// checkRecord loads one record. wantDefaultCeremony holds only for a
	// record that carries no ceremony field. A live record can carry an
	// explicit level, so the caller decides.
	checkRecord := func(t *testing.T, root, changeID string, wantDefaultCeremony bool) {
		t.Helper()

		record, err := Load(root, changeID)
		if err != nil {
			t.Fatalf("Load(%s) error = %v", changeID, err)
		}
		if record.ChangeID != changeID {
			t.Fatalf("ChangeID = %q, want %q", record.ChangeID, changeID)
		}
		if wantDefaultCeremony && record.CeremonyLevel() != DefaultCeremony {
			t.Fatalf("CeremonyLevel() = %d, want %d", record.CeremonyLevel(), DefaultCeremony)
		}
	}

	t.Run("fixtures", func(t *testing.T) {
		t.Parallel()

		entries, err := os.ReadDir(filepath.Join("testdata", "v0"))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) == 0 {
			t.Fatal("testdata/v0 holds no record")
		}

		root := t.TempDir()
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			changeID := entry.Name()
			source := filepath.Join("testdata", "v0", changeID, "change.json")
			data, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, ".shipproof", "changes", changeID)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "change.json"), data, 0o644); err != nil {
				t.Fatal(err)
			}
			checkRecord(t, root, changeID, true)
		}
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()

		root := filepath.Join("..", "..")
		entries, err := os.ReadDir(filepath.Join(root, ".shipproof", "changes"))
		if err != nil {
			t.Skipf("no live change directory: %v", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			checkRecord(t, root, entry.Name(), false)
		}
	})
}
