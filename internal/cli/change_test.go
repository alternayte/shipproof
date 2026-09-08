package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangeStart(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	src := filepath.Join(root, "change-src.md")
	if err := os.WriteFile(src, []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"start", "SP-030", "--intent", src}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	recordPath := filepath.Join(root, ".shipproof", "changes", "SP-030", "change.json")
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("read change record: %v", err)
	}
	var record struct {
		ChangeID   string `json:"change_id"`
		ShapingRef string `json:"shaping_ref"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("parse change record: %v", err)
	}
	if record.ChangeID != "SP-030" {
		t.Errorf("change_id = %q, want SP-030", record.ChangeID)
	}
	if record.ShapingRef != "" {
		t.Errorf("the record must hold no shaping reference, got %q", record.ShapingRef)
	}
}

func TestChangeStartRejectsShapingFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"start", "SP-032", "--intent", "x.md", "--shaping", "my-session"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestChangeStartForceResnapshotsAndKeepsCeremony(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	src := filepath.Join(root, "SP-040.md")
	if err := os.WriteFile(src, []byte("# SP-040\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"start", "SP-040", "--intent", src, "--ceremony", "2"}, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0, got stderr %s", stderr.String())
	}

	if err := os.WriteFile(src, []byte("# SP-040\n\nnew intent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"start", "SP-040", "--intent", src, "--force"}, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0 with --force, got stderr %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Ceremony: 2") {
		t.Fatalf("--force did not keep the ceremony level:\n%s", stdout.String())
	}

	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "changes", "SP-040", "snapshot.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "new intent") {
		t.Fatalf("--force did not re-snapshot the source document:\n%s", data)
	}
}

func TestChangeStartWithoutForceRefusesAnExistingChange(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	src := filepath.Join(root, "SP-041.md")
	if err := os.WriteFile(src, []byte("# SP-041\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"start", "SP-041", "--intent", src}, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0, got stderr %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"start", "SP-041", "--intent", src}, stdout, stderr); code != 1 {
		t.Fatalf("expected exit 1 for an existing change, got %d", code)
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("stderr does not carry the refusal:\n%s", stderr.String())
	}
}

func TestChangeStartRejectsCeremonyOutOfRangeWithExitTwo(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	src := filepath.Join(root, "SP-042.md")
	if err := os.WriteFile(src, []byte("# SP-042\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := Run([]string{"start", "SP-042", "--intent", src, "--ceremony", "4"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %s", code, stderr.String())
	}
}
