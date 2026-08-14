package shaping

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartAndLoad(t *testing.T) {
	root := t.TempDir()
	session, path, err := Start(root, StartOptions{Kind: "prd", Subject: "Webhook retries"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if session.SessionID != "webhook-retries" {
		t.Fatalf("session id = %q", session.SessionID)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file: %v", err)
	}

	loaded, _, err := Load(root, "webhook-retries")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Subject != "Webhook retries" {
		t.Fatalf("subject = %q", loaded.Subject)
	}
}

func TestStartDoesNotOverwrite(t *testing.T) {
	root := t.TempDir()
	if _, _, err := Start(root, StartOptions{Kind: "sdd", Subject: "Rotation"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Start(root, StartOptions{Kind: "sdd", Subject: "Rotation"}); err == nil {
		t.Fatal("expected duplicate session error")
	}
}

func TestCheckFileRejectsInconsistentReadyState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	contents := `{
  "schema_version":"0.1",
  "subject":"Test",
  "document_kind":"prd",
  "state":"ready",
  "current_model":{},
  "decisions":[],
  "assumptions":[{"id":"A1","summary":"Unknown traffic"}],
  "risks":[],
  "unknowns":[],
  "readiness":{"blockers":[],"decisions_required":[]}
}`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckFile(path); err == nil {
		t.Fatal("expected readiness consistency error")
	}
}
