package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewPrepare(t *testing.T) {
	root := t.TempDir()
	setupReviewTestRepo(t, root)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"review", "prepare", "SP-006"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Review packet written:") {
		t.Errorf("expected success message, got: %s", output)
	}

	packetPath := filepath.Join(root, ".shipproof", "changes", "SP-006", "review-packet.json")
	data, err := os.ReadFile(packetPath)
	if err != nil {
		t.Fatalf("review-packet.json not written: %v", err)
	}

	var packet struct {
		ChangeID string `json:"change_id"`
	}
	if err := json.Unmarshal(data, &packet); err != nil {
		t.Fatalf("parse review packet: %v", err)
	}
	if packet.ChangeID != "SP-006" {
		t.Errorf("expected change_id SP-006, got %s", packet.ChangeID)
	}
}

func TestReviewPrepareMissingChangeID(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"review", "prepare"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestReviewPrepareMissingChange(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	RunOverrides["."] = root
	defer delete(RunOverrides, ".")

	code := Run([]string{"review", "prepare", "SP-999"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
}

func TestReviewPrepareUnknownSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"review", "unknown"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func setupReviewTestRepo(t *testing.T, root string) {
	t.Helper()
	setupShipProofDir(t, root)

	changeDir := filepath.Join(root, ".shipproof", "changes", "SP-006")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}

	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      "SP-006",
		"source_path":    "docs/changes/SP-006-human-review-packet.md",
		"snapshot_path":  ".shipproof/changes/SP-006/snapshot.md",
		"sha256":         "abc123",
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}

	evidencePack := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      "SP-006",
		"intent": map[string]interface{}{
			"snapshot_hash": "abc123",
			"requirements": []map[string]interface{}{
				{"id": "R1", "verification_refs": []string{"go test"}},
			},
		},
		"verification": map[string]interface{}{
			"checks": []map[string]interface{}{
				{"id": "pass:check", "status": "pass", "source": "runner", "provenance": "observed"},
				{"id": "fail:check", "status": "fail", "source": "runner", "provenance": "observed"},
				{"id": "unknown:check", "status": "unknown", "source": "runner", "provenance": "observed"},
			},
		},
		"implementation": map[string]interface{}{
			"commits":       []map[string]string{},
			"changed_files": []string{},
			"additions":     0,
			"deletions":     0,
			"diff_stat":     "",
		},
		"provenance": map[string]string{
			"generated_at":      "2024-01-01T00:00:00Z",
			"shipproof_version": "0.1",
		},
	}
	data, _ = json.MarshalIndent(evidencePack, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "evidence-pack.json"), data, 0o644); err != nil {
		t.Fatalf("write evidence pack: %v", err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
}
