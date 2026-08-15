package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shipproof/shipproof/internal/schema"
)

func TestReportChangeWritesHTML(t *testing.T) {
	root, changeID := setupReportCLITest(t)
	defer os.RemoveAll(root)

	setupCLIEvidencePack(t, root, changeID)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "change", changeID}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit code %d: %s", exit, stderr.String())
	}

	html := stdout.String()
	if !hasPrefix(html, "<!DOCTYPE html>") {
		t.Error("stdout should start with <!DOCTYPE html>")
	}
	if !contains(html, changeID) {
		t.Error("stdout should contain change ID")
	}
}

func TestReportChangeMissingArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "change"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("expected exit code 2 for missing args, got %d", exit)
	}
}

func TestReportChangeMissingEvidencePack(t *testing.T) {
	root, changeID := setupReportCLITest(t)
	defer os.RemoveAll(root)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "change", changeID}, &stdout, &stderr)
	if exit != 1 {
		t.Fatalf("expected exit code 1 for missing evidence pack, got %d", exit)
	}
}

func TestReportOutputFlag(t *testing.T) {
	root, changeID := setupReportCLITest(t)
	defer os.RemoveAll(root)

	setupCLIEvidencePack(t, root, changeID)

	outputPath := filepath.Join(root, "output.html")
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "change", changeID, "--output", outputPath}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit code %d: %s", exit, stderr.String())
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file not written: %v", err)
	}
}

func TestReportPRSummaryWritesMarkdown(t *testing.T) {
	root, changeID := setupReportCLITest(t)
	defer os.RemoveAll(root)

	setupCLIEvidencePack(t, root, changeID)
	setupCLIReviewPacket(t, root, changeID)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "pr-summary", changeID}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit code %d: %s", exit, stderr.String())
	}

	md := stdout.String()
	if !contains(md, "## What changed") {
		t.Error("stdout should contain What changed section")
	}
	if !contains(md, "## Deterministic evidence") {
		t.Error("stdout should contain Deterministic evidence section")
	}
}

func TestReportPRSummaryMissingArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "pr-summary"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("expected exit code 2 for missing args, got %d", exit)
	}
}

func TestReportUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "unknown-sub"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("expected exit code 2 for unknown subcommand, got %d", exit)
	}
}

func TestReportMissingArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("expected exit code 2 for missing subcommand, got %d", exit)
	}
}

func setupReportCLITest(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	changeID := "SP-CLI"

	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatalf("create .shipproof: %v", err)
	}

	changeDir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}

	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      changeID,
		"source_path":    "docs/changes/test.md",
		"snapshot_path":  ".shipproof/changes/" + changeID + "/snapshot.md",
		"sha256":         "abc123",
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(changeDir, "change.json"), data, 0o644)

	origDir, _ := os.Getwd()
	os.Chdir(root)
	t.Cleanup(func() { os.Chdir(origDir) })

	return root, changeID
}

func TestReportProjectWritesHTML(t *testing.T) {
	root, changeID := setupReportCLITest(t)
	defer os.RemoveAll(root)

	setupCLIEvidencePack(t, root, changeID)

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "project", "test-proj"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit code %d: %s", exit, stderr.String())
	}

	html := stdout.String()
	if !hasPrefix(html, "<!DOCTYPE html>") {
		t.Error("stdout should start with <!DOCTYPE html>")
	}
	if !contains(html, "test-proj") {
		t.Error("stdout should contain project name")
	}
}

func TestReportProjectOutputFlag(t *testing.T) {
	root, changeID := setupReportCLITest(t)
	defer os.RemoveAll(root)

	setupCLIEvidencePack(t, root, changeID)

	outputPath := filepath.Join(root, "project-report.html")
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "project", "test-proj", "--output", outputPath}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit code %d: %s", exit, stderr.String())
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file not written: %v", err)
	}
}

func TestReportProjectMissingArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"report", "project"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("expected exit code 2 for missing project name, got %d", exit)
	}
}

func setupCLIEvidencePack(t *testing.T, root, changeID string) {
	t.Helper()

	ev := schema.EvidencePack{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
		Intent: schema.IntentEvidence{
			SnapshotHash: "abc123",
			Requirements: []schema.Requirement{
				{ID: "R1", VerificationRefs: []string{"go test -run TestR1"}},
			},
		},
		Implementation: schema.ImplementationEvidence{
			Commits: []schema.ImplementationCommit{
				{Hash: "abcdef123456", Author: "Dev", Subject: "Add feature"},
			},
			ChangedFiles: []string{"src/main.go"},
			Additions:    10,
			Deletions:    0,
			DiffStat:     "1 file changed",
		},
		Verification: schema.VerificationEvidence{
			Checks: []schema.Check{
				{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
			},
		},
		Provenance: schema.PackProvenance{
			GeneratedAt:      "2026-08-14T20:00:00Z",
			ShipProofVersion: "0.1",
		},
	}

	changeDir := filepath.Join(root, ".shipproof", "changes", changeID)
	data, _ := json.MarshalIndent(ev, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(changeDir, "evidence-pack.json"), data, 0o644)
}

func setupCLIReviewPacket(t *testing.T, root, changeID string) {
	t.Helper()

	packet := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      changeID,
		"intent": map[string]interface{}{
			"snapshot_hash":     "abc123",
			"requirement_count": 1,
		},
		"git_summary": map[string]interface{}{
			"commits":             []interface{}{},
			"changed_files_count": 1,
			"additions":           10,
			"deletions":           0,
		},
		"already_proven": []map[string]string{
			{"id": "check:unit", "status": "pass", "source": "junit", "provenance": "observed"},
		},
		"human_attention": []interface{}{},
		"unknown":         []interface{}{},
	}

	changeDir := filepath.Join(root, ".shipproof", "changes", changeID)
	data, _ := json.MarshalIndent(packet, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(changeDir, "review-packet.json"), data, 0o644)
}

func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && stringContainsStr(s, sub)
}

func stringContainsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
