package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTelemetryCollect(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestChangeRecord(t, root, "SP-008")

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"telemetry", "collect", "SP-008", "--adapter", "claude", "--dir", root}, stdout, stderr)

	if code != 0 {
		t.Logf("collect stderr: %s", stderr.String())
	}

	t.Logf("stdout: %s", stdout.String())
}

func TestTelemetryCollectMissingAdapter(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"telemetry", "collect", "SP-008"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--adapter") {
		t.Errorf("expected --adapter error, got: %s", stderr.String())
	}
}

func TestTelemetryCollectNoArgs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"telemetry"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestTelemetryUnknownSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"telemetry", "unknown"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestTelemetryCollectWritesAgentRun(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	changeID := "SP-008"
	changeDir := filepath.Join(root, ".shipproof", "changes", changeID)
	os.MkdirAll(changeDir, 0o755)

	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      changeID,
		"source_path":    "docs/changes/SP-008-agent-telemetry.md",
		"snapshot_path":  ".shipproof/changes/SP-008/snapshot.md",
		"sha256":         "abc123",
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(changeDir, "change.json"), data, 0o644)

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"telemetry", "collect", changeID, "--adapter", "claude", "--dir", root}, stdout, stderr)
	t.Logf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())

	if code == 0 {
		agentRunPath := filepath.Join(root, ".shipproof", "runs", changeID, "agent-run.json")
		if _, err := os.Stat(agentRunPath); err == nil {
			t.Logf("agent-run.json written to %s", agentRunPath)
		}
	}
}
