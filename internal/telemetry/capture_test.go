package telemetry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
)

func writeEvidenceConfig(t *testing.T, root, capture string) {
	t.Helper()
	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "evidence:\n  capture: " + capture + "\n"
	if err := os.WriteFile(filepath.Join(shipproofDir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyCaptureLevelMetadataKeepsReferenceOnly(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeEvidenceConfig(t, root, "metadata")

	rawFile := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"type":"user","message":"hello"}
`
	if err := os.WriteFile(rawFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	runDir := filepath.Join(root, ".shipproof", "runs", "SP-018")
	run := agent.AgentRun{}
	adapter := &fakeRawProvider{name: "fake", rawPath: rawFile}

	if err := applyCaptureLevel(root, runDir, adapter, root, &run); err != nil {
		t.Fatalf("applyCaptureLevel() error = %v", err)
	}

	if run.RawLogRef != rawFile {
		t.Errorf("RawLogRef = %q, want the original path %q", run.RawLogRef, rawFile)
	}

	if _, err := os.Stat(filepath.Join(runDir, "agent-raw")); !errors.Is(err, os.ErrNotExist) {
		t.Error("metadata capture must not copy transcripts")
	}
}

func TestApplyCaptureLevelRedactedCopiesWithMasking(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeEvidenceConfig(t, root, "redacted")

	rawFile := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"type":"assistant","message":"token ghp_abcdefghijklmnopqrstuvwxyz012345 was used"}
`
	if err := os.WriteFile(rawFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	runDir := filepath.Join(root, ".shipproof", "runs", "SP-018")
	run := agent.AgentRun{}
	adapter := &fakeRawProvider{name: "fake", rawPath: rawFile}

	if err := applyCaptureLevel(root, runDir, adapter, root, &run); err != nil {
		t.Fatalf("applyCaptureLevel() error = %v", err)
	}

	if !strings.HasPrefix(run.RawLogRef, ".shipproof/runs/SP-018/agent-raw") {
		t.Errorf("RawLogRef = %q, want a captured path under agent-raw", run.RawLogRef)
	}

	copied, err := os.ReadFile(filepath.Join(runDir, "agent-raw", "session.jsonl"))
	if err != nil {
		t.Fatalf("read captured transcript: %v", err)
	}
	if strings.Contains(string(copied), "ghp_abcdefghijklmnopqrstuvwxyz012345") {
		t.Error("expected secrets masked in the redacted copy")
	}
	if !strings.Contains(string(copied), "[REDACTED]") {
		t.Error("expected a redaction marker in the redacted copy")
	}
}

func TestApplyCaptureLevelFullCopiesVerbatim(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeEvidenceConfig(t, root, "full")

	rawFile := filepath.Join(t.TempDir(), "session.jsonl")
	content := `{"type":"assistant","message":"plain content"}
`
	if err := os.WriteFile(rawFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	runDir := filepath.Join(root, ".shipproof", "runs", "SP-018")
	run := agent.AgentRun{}
	adapter := &fakeRawProvider{name: "fake", rawPath: rawFile}

	if err := applyCaptureLevel(root, runDir, adapter, root, &run); err != nil {
		t.Fatalf("applyCaptureLevel() error = %v", err)
	}

	copied, err := os.ReadFile(filepath.Join(runDir, "agent-raw", "session.jsonl"))
	if err != nil {
		t.Fatalf("read captured transcript: %v", err)
	}
	if string(copied) != content {
		t.Errorf("full capture must copy verbatim, got %q", string(copied))
	}
}

func TestApplyCaptureLevelMissingRawLogIsIgnored(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeEvidenceConfig(t, root, "full")

	runDir := filepath.Join(root, ".shipproof", "runs", "SP-018")
	run := agent.AgentRun{}
	adapter := &fakeRawProvider{name: "fake"}

	if err := applyCaptureLevel(root, runDir, adapter, root, &run); err != nil {
		t.Fatalf("applyCaptureLevel() error = %v", err)
	}
	if run.RawLogRef != "" {
		t.Errorf("RawLogRef = %q, want empty when no raw log exists", run.RawLogRef)
	}
}

func TestApplyCaptureLevelMissingConfigDefaultsToMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755); err != nil {
		t.Fatal(err)
	}

	rawFile := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(rawFile, []byte(`{"message":"hello"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runDir := filepath.Join(root, ".shipproof", "runs", "SP-018")
	run := agent.AgentRun{}
	adapter := &fakeRawProvider{name: "fake", rawPath: rawFile}

	if err := applyCaptureLevel(root, runDir, adapter, root, &run); err != nil {
		t.Fatalf("applyCaptureLevel() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(runDir, "agent-raw")); !errors.Is(err, os.ErrNotExist) {
		t.Error("missing config must default to metadata and not copy transcripts")
	}
}

func TestApplyCaptureLevelInvalidConfigFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeEvidenceConfig(t, root, "everything")

	run := agent.AgentRun{}
	adapter := &fakeRawProvider{name: "fake"}

	if err := applyCaptureLevel(root, filepath.Join(root, ".shipproof", "runs", "SP-018"), adapter, root, &run); err == nil {
		t.Fatal("expected error for invalid capture level")
	}
}
