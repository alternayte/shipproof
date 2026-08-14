package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidencePack(t *testing.T) {
	root := t.TempDir()
	setupTestRepo(t, root)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "pack", "SP-005"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Evidence pack written:") {
		t.Errorf("expected success message, got: %s", output)
	}

	packPath := filepath.Join(root, ".shipproof", "changes", "SP-005", "evidence-pack.json")
	data, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("evidence-pack.json not written: %v", err)
	}

	var pack struct {
		ChangeID string `json:"change_id"`
	}
	if err := json.Unmarshal(data, &pack); err != nil {
		t.Fatalf("parse evidence pack: %v", err)
	}
	if pack.ChangeID != "SP-005" {
		t.Errorf("expected change_id SP-005, got %s", pack.ChangeID)
	}
}

func TestEvidencePackMissingChangeID(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "pack"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestEvidencePackMissingChange(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	RunOverrides["."] = root
	defer delete(RunOverrides, ".")

	code := Run([]string{"evidence", "pack", "SP-999"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
}

func TestEvidencePackMissingVerificationPlan(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestChangeRecord(t, root, "SP-099")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	RunOverrides["."] = root
	defer delete(RunOverrides, ".")

	code := Run([]string{"evidence", "pack", "SP-099"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func setupTestRepo(t *testing.T, root string) {
	t.Helper()
	setupShipProofDir(t, root)

	changeDir := filepath.Join(root, ".shipproof", "changes", "SP-005")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}

	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      "SP-005",
		"source_path":    "docs/changes/SP-005-test.md",
		"snapshot_path":  ".shipproof/changes/SP-005/snapshot.md",
		"sha256":         "abc123",
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}

	plan := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      "SP-005",
		"requirements": []map[string]interface{}{
			{
				"id":        "SP-005-R1",
				"statement": "Load intent snapshot metadata.",
				"proof": []map[string]string{
					{"type": "unit", "target": "test.go", "command": "go test"},
				},
			},
		},
		"invariants": []map[string]interface{}{
			{
				"id":        "INV-TEST",
				"statement": "Must be valid.",
				"proof": []map[string]string{
					{"type": "unit", "target": "test.go", "command": "go test"},
				},
			},
		},
	}
	data, _ = json.MarshalIndent(plan, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "verification.json"), data, 0o644); err != nil {
		t.Fatalf("write verification plan: %v", err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
}

func setupShipProofDir(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create .shipproof: %v", err)
	}
}

func writeTestChangeRecord(t *testing.T, root, changeID string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}
	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      changeID,
		"source_path":    "docs/changes/" + changeID + "-test.md",
		"snapshot_path":  ".shipproof/changes/" + changeID + "/snapshot.md",
		"sha256":         "abc123",
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}
}
