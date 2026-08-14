package opencode

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeAdapter(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "opencode.db")

	setupTestDB(t, dbPath, "/Users/test/project")

	adapter := NewAdapterWithDB(dbPath)
	run, err := adapter.Collect("/Users/test/project")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if run.SessionID != "ses_test123" {
		t.Errorf("session_id = %q, want ses_test123", run.SessionID)
	}
	if run.AgentVersion != "1.2.3" {
		t.Errorf("agent_version = %q, want 1.2.3", run.AgentVersion)
	}
	if run.Model != "deepseek-v4-pro" {
		t.Errorf("model = %q, want deepseek-v4-pro", run.Model)
	}
	if run.Cost <= 0 {
		t.Errorf("cost = %f, want positive value", run.Cost)
	}
	if run.Tokens == nil {
		t.Fatal("tokens must not be nil")
	}
	if run.Tokens.Input <= 0 {
		t.Errorf("tokens.input = %d, want positive value", run.Tokens.Input)
	}
	if run.Tokens.Output <= 0 {
		t.Errorf("tokens.output = %d, want positive value", run.Tokens.Output)
	}
	if run.StartedAt == "" {
		t.Error("started_at is empty")
	}
	if run.EndedAt == "" {
		t.Error("ended_at is empty")
	}
}

func TestOpenCodeAdapterNoDB(t *testing.T) {
	adapter := NewAdapterWithDB("/nonexistent/path.db")
	_, err := adapter.Collect("/some/project")
	if err == nil {
		t.Fatal("expected error for missing database")
	}
	if !strings.Contains(err.Error(), "database not found") {
		t.Errorf("error = %v, want 'database not found'", err)
	}
}

func TestOpenCodeAdapterNoSession(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "empty.db")

	setupTestDB(t, dbPath, "/Users/test/other")

	adapter := NewAdapterWithDB(dbPath)
	_, err := adapter.Collect("/Users/test/missing-project")
	if err == nil {
		t.Fatal("expected error for no session")
	}
	if !strings.Contains(err.Error(), "no session found") {
		t.Errorf("error = %v, want 'no session found'", err)
	}
}

func TestOpenCodeAdapterZeroTokens(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "zerotokens.db")

	setupTestDBWithTokens(t, dbPath, "/Users/test/project", 0, 0)

	adapter := NewAdapterWithDB(dbPath)
	run, err := adapter.Collect("/Users/test/project")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if run.Tokens != nil {
		t.Error("tokens must be nil when both input and output are zero")
	}
}

func TestOpenCodeAdapterPicksMostRecent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "multisession.db")

	setupTestDBWithTwoSessions(t, dbPath, "/Users/test/project")

	adapter := NewAdapterWithDB(dbPath)
	run, err := adapter.Collect("/Users/test/project")
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if run.SessionID != "ses_newer" {
		t.Errorf("session_id = %q, want ses_newer (most recent)", run.SessionID)
	}
}

func setupTestDB(t *testing.T, dbPath, projectDir string) {
	t.Helper()
	setupTestDBWithTokens(t, dbPath, projectDir, 50000, 8000)
}

func setupTestDBWithTokens(t *testing.T, dbPath, projectDir string, inputTokens, outputTokens int64) {
	t.Helper()

	sql := fmt.Sprintf(`
		CREATE TABLE project (
			id TEXT PRIMARY KEY,
			worktree TEXT NOT NULL,
			vcs TEXT,
			name TEXT,
			icon_url TEXT,
			icon_url_override TEXT,
			icon_color TEXT,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			time_initialized INTEGER,
			sandboxes TEXT NOT NULL,
			commands TEXT
		);

		CREATE TABLE session (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			workspace_id TEXT,
			parent_id TEXT,
			slug TEXT NOT NULL,
			directory TEXT NOT NULL,
			path TEXT,
			title TEXT NOT NULL,
			version TEXT NOT NULL,
			share_url TEXT,
			summary_additions INTEGER,
			summary_deletions INTEGER,
			summary_files INTEGER,
			summary_diffs TEXT,
			metadata TEXT,
			cost REAL DEFAULT 0 NOT NULL,
			tokens_input INTEGER DEFAULT 0 NOT NULL,
			tokens_output INTEGER DEFAULT 0 NOT NULL,
			tokens_reasoning INTEGER DEFAULT 0 NOT NULL,
			tokens_cache_read INTEGER DEFAULT 0 NOT NULL,
			tokens_cache_write INTEGER DEFAULT 0 NOT NULL,
			revert TEXT,
			permission TEXT,
			agent TEXT,
			model TEXT,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			time_compacting INTEGER,
			time_archived INTEGER,
			FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
		);

		INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes)
		VALUES ('proj_abc123', '%s', 'git', 'test-project', 1000000, 2000000, '[]');

		INSERT INTO session (id, project_id, slug, directory, title, version, cost, tokens_input, tokens_output, agent, model, time_created, time_updated)
		VALUES ('ses_test123', 'proj_abc123', 'test-session', '%s', 'Test session', '1.2.3', 0.42, %d, %d, 'build', '{"id":"deepseek-v4-pro","providerID":"opencode","variant":"max"}', 3000000, 4000000);
	`, escapeSQL(projectDir), escapeSQL(projectDir), inputTokens, outputTokens)

	cmd := exec.Command("sqlite3", dbPath)
	in, _ := cmd.StdinPipe()
	cmd.Start()
	in.Write([]byte(sql))
	in.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("setup test database: %v", err)
	}
}

func setupTestDBWithTwoSessions(t *testing.T, dbPath, projectDir string) {
	t.Helper()

	sql := fmt.Sprintf(`
		CREATE TABLE project (
			id TEXT PRIMARY KEY,
			worktree TEXT NOT NULL,
			vcs TEXT,
			name TEXT,
			icon_url TEXT,
			icon_url_override TEXT,
			icon_color TEXT,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			time_initialized INTEGER,
			sandboxes TEXT NOT NULL,
			commands TEXT
		);

		CREATE TABLE session (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			workspace_id TEXT,
			parent_id TEXT,
			slug TEXT NOT NULL,
			directory TEXT NOT NULL,
			title TEXT NOT NULL,
			version TEXT NOT NULL,
			share_url TEXT,
			summary_additions INTEGER,
			summary_deletions INTEGER,
			summary_files INTEGER,
			summary_diffs TEXT,
			metadata TEXT,
			cost REAL DEFAULT 0 NOT NULL,
			tokens_input INTEGER DEFAULT 0 NOT NULL,
			tokens_output INTEGER DEFAULT 0 NOT NULL,
			tokens_reasoning INTEGER DEFAULT 0 NOT NULL,
			tokens_cache_read INTEGER DEFAULT 0 NOT NULL,
			tokens_cache_write INTEGER DEFAULT 0 NOT NULL,
			revert TEXT,
			permission TEXT,
			agent TEXT,
			model TEXT,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			time_compacting INTEGER,
			time_archived INTEGER,
			FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
		);

		INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes)
		VALUES ('proj_multi', '%s', 'git', 'multi-project', 1000000, 2000000, '[]');

		INSERT INTO session (id, project_id, slug, directory, title, version, cost, tokens_input, tokens_output, agent, model, time_created, time_updated)
		VALUES ('ses_older', 'proj_multi', 'older', '%s', 'Older', '1.0.0', 0.10, 1000, 200, 'build', '{"id":"gpt-4","providerID":"openai"}', 1000000, 2000000);

		INSERT INTO session (id, project_id, slug, directory, title, version, cost, tokens_input, tokens_output, agent, model, time_created, time_updated)
		VALUES ('ses_newer', 'proj_multi', 'newer', '%s', 'Newer', '2.0.0', 0.25, 3000, 500, 'build', '{"id":"claude-sonnet","providerID":"anthropic"}', 3000000, 4000000);
	`, escapeSQL(projectDir), escapeSQL(projectDir), escapeSQL(projectDir))

	cmd := exec.Command("sqlite3", dbPath)
	in, _ := cmd.StdinPipe()
	cmd.Start()
	in.Write([]byte(sql))
	in.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("setup test database: %v", err)
	}
}
