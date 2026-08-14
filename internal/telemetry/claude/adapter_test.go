package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeAdapter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectDir := "/Users/test/project"

	sessions := []claudeSession{
		{
			PID:       100,
			SessionID: "old-session",
			CWD:       projectDir,
			StartedAt: 1000,
			Version:   "1.0.0",
			Status:    "idle",
			UpdatedAt: 2000,
		},
		{
			PID:       200,
			SessionID: "session-abc-123",
			CWD:       projectDir,
			StartedAt: 3000,
			Version:   "2.0.0",
			Status:    "completed",
			UpdatedAt: 5000,
		},
	}

	for i, s := range sessions {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(dir, s.SessionID+".json")
		if err := os.WriteFile(name, data, 0o644); err != nil {
			t.Fatal(err)
		}
		_ = i
	}

	adapter := NewAdapterWithDir(dir)
	run, err := adapter.Collect(projectDir)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if run.SessionID != "session-abc-123" {
		t.Errorf("session_id = %q, want session-abc-123", run.SessionID)
	}
	if run.AgentVersion != "2.0.0" {
		t.Errorf("agent_version = %q, want 2.0.0", run.AgentVersion)
	}
	if run.ExitStatus != "completed" {
		t.Errorf("exit_status = %q, want completed", run.ExitStatus)
	}
	if run.StartedAt == "" {
		t.Error("started_at is empty")
	}
	if run.EndedAt == "" {
		t.Error("ended_at is empty")
	}
}

func TestClaudeAdapterNoSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	adapter := NewAdapterWithDir(dir)

	_, err := adapter.Collect("/Users/test/missing-project")
	if err == nil {
		t.Fatal("expected error for no sessions")
	}
	if !strings.Contains(err.Error(), "no session found") {
		t.Errorf("error = %v, want 'no session found'", err)
	}
}

func TestClaudeAdapterSkipsOtherProjects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	otherProject := "/Users/test/other"
	targetProject := "/Users/test/target"

	s := claudeSession{
		PID:       300,
		SessionID: "other-session",
		CWD:       otherProject,
		Version:   "3.0.0",
		UpdatedAt: 9000,
	}
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(dir, "other.json"), data, 0o644)

	s2 := claudeSession{
		PID:       400,
		SessionID: "target-session",
		CWD:       targetProject,
		Version:   "4.0.0",
		UpdatedAt: 8000,
	}
	data2, _ := json.Marshal(s2)
	os.WriteFile(filepath.Join(dir, "target.json"), data2, 0o644)

	adapter := NewAdapterWithDir(dir)
	run, err := adapter.Collect(targetProject)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if run.SessionID != "target-session" {
		t.Errorf("session_id = %q, want target-session", run.SessionID)
	}
}

func TestClaudeAdapterMissingFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	projectDir := "/Users/test/project"

	s := claudeSession{
		PID:       500,
		SessionID: "minimal-session",
		CWD:       projectDir,
		Version:   "",
		UpdatedAt: 100,
	}
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(dir, "minimal.json"), data, 0o644)

	adapter := NewAdapterWithDir(dir)
	run, err := adapter.Collect(projectDir)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if run.Cost != 0 {
		t.Error("cost must be zero for claude adapter")
	}
	if run.Tokens != nil {
		t.Error("tokens must be nil for claude adapter")
	}
	if run.Model == "" {
		t.Log("model is empty (settings.json not available)")
	}
}
