package verify

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunEcho(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "echo hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0", result.ExitCode)
	}
	if result.DurationMs == 0 {
		t.Fatal("duration_ms is 0")
	}

	stdout, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.StdoutPath)))
	if err != nil {
		t.Fatal(err)
	}
	if string(stdout) != "hello\n" {
		t.Fatalf("stdout = %q, want %q", string(stdout), "hello\n")
	}
}

func TestRunSuccess(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0", result.ExitCode)
	}
}

func TestRunFailureExitCode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "exit 42")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 42 {
		t.Fatalf("exit_code = %d, want 42", result.ExitCode)
	}
}

func TestRunFailureNoCrash(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "exit 1")
	if err != nil {
		t.Fatalf("Run() must not return error on non-zero exit, got %v", err)
	}
	if result.ExitCode != 1 {
		t.Fatalf("exit_code = %d, want 1", result.ExitCode)
	}
}

func TestRunHasDuration(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "sleep 0.1")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.DurationMs < 50 {
		t.Fatalf("duration_ms = %d, expected at least 50ms", result.DurationMs)
	}
}

func TestRunCapturesStdout(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "echo captured-output")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	stdout, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.StdoutPath)))
	if err != nil {
		t.Fatal(err)
	}
	if string(stdout) != "captured-output\n" {
		t.Fatalf("stdout = %q", string(stdout))
	}
}

func TestRunCapturesStderr(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "echo error-output >&2")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	stderr, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.StderrPath)))
	if err != nil {
		t.Fatal(err)
	}
	if string(stderr) != "error-output\n" {
		t.Fatalf("stderr = %q", string(stderr))
	}
}

func TestRunRejectsEmptyChangeID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := Run(root, "", "true")
	if err == nil {
		t.Fatal("expected error for empty change id")
	}
}

func TestRunRejectsEmptyCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	_, err := Run(root, "SP-002", "")
	if !errors.Is(err, ErrCommandMissing) {
		t.Fatalf("expected ErrCommandMissing, got %v", err)
	}
}

func TestRunCreatesOutputDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	runDir := filepath.Join(root, ".shipproof", "runs", "SP-002")
	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Fatal("run directory must not exist before Run")
	}

	_, err := Run(root, "SP-002", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Stat(runDir); err != nil {
		t.Fatal("run directory was not created")
	}
}

func TestRunWritesResultJSON(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	runPath := filepath.Join(root, ".shipproof", "runs", "SP-002", "run.json")
	data, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("run.json is empty")
	}

	if result.SchemaVersion != "0.1" {
		t.Fatalf("schema_version = %q, want 0.1", result.SchemaVersion)
	}
	if result.ChangeID != "SP-002" {
		t.Fatalf("change_id = %q, want SP-002", result.ChangeID)
	}
	if result.Timestamp == "" {
		t.Fatal("timestamp is empty")
	}
}

func TestRunWithArgs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := Run(root, "SP-002", "printf '%s %s %s' one two three")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	stdout, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(result.StdoutPath)))
	if err != nil {
		t.Fatal(err)
	}
	if string(stdout) != "one two three" {
		t.Fatalf("stdout = %q, want %q", string(stdout), "one two three")
	}
}

func setupRepo(t *testing.T, root string) {
	t.Helper()
	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configContent := "version: 1\nverification:\n  command: just verify\n"
	if err := os.WriteFile(filepath.Join(shipproofDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}
	changesDir := filepath.Join(shipproofDir, "changes", "SP-002")
	if err := os.MkdirAll(changesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	changeContent := `{
  "schema_version": "0.1",
  "change_id": "SP-002",
  "source_path": "docs/changes/SP-002-verification-runner.md",
  "snapshot_path": ".shipproof/changes/SP-002/snapshot.md",
  "sha256": "abc123",
  "captured_at": "2026-08-14T21:06:03Z"
}
`
	if err := os.WriteFile(filepath.Join(changesDir, "change.json"), []byte(changeContent), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunAdhoc(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := RunAdhoc(root, "echo adhoc")
	if err != nil {
		t.Fatalf("RunAdhoc() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0", result.ExitCode)
	}

	logPath := filepath.Join(root, ".shipproof", "runs", "adhoc", "stdout.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read adhoc stdout: %v", err)
	}
	if string(content) != "adhoc\n" {
		t.Fatalf("stdout = %q, want %q", string(content), "adhoc\n")
	}

	runJSON := filepath.Join(root, ".shipproof", "runs", "adhoc", "run.json")
	if _, err := os.Stat(runJSON); !errors.Is(err, os.ErrNotExist) {
		t.Error("expected no run.json for an adhoc run")
	}
}

func TestRunAdhocFailureExitCode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	result, err := RunAdhoc(root, "exit 3")
	if err != nil {
		t.Fatalf("RunAdhoc() error = %v", err)
	}
	if result.ExitCode != 3 {
		t.Fatalf("exit_code = %d, want 3", result.ExitCode)
	}
}

func TestRunAdhocMissingCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	setupRepo(t, root)

	if _, err := RunAdhoc(root, "  "); !errors.Is(err, ErrCommandMissing) {
		t.Fatalf("expected ErrCommandMissing, got %v", err)
	}
}

func TestRunRecordsHeadRevisionAndCleanTree(t *testing.T) {
	root := initVerifyTestRepo(t)

	result, err := Run(root, "SP-200", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.HeadRev == "" {
		t.Fatal("HeadRev is empty; want the current revision")
	}
	if result.TreeClean == nil || !*result.TreeClean {
		t.Fatalf("TreeClean = %v, want a pointer to true", result.TreeClean)
	}
}

func TestRunRecordsDirtyTree(t *testing.T) {
	root := initVerifyTestRepo(t)
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(root, "SP-201", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.TreeClean == nil || *result.TreeClean {
		t.Fatalf("TreeClean = %v, want a pointer to false", result.TreeClean)
	}
}

func TestRunIgnoresShipproofWrites(t *testing.T) {
	root := initVerifyTestRepo(t)
	if err := os.MkdirAll(filepath.Join(root, ".shipproof", "changes", "SP-203"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".shipproof", "changes", "SP-203", "evidence-pack.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(root, "SP-203", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.TreeClean == nil || !*result.TreeClean {
		t.Fatalf("TreeClean = %v, want a pointer to true; a ShipProof write must not dirty the tree", result.TreeClean)
	}
}

func TestRunOutsideGitRepositoryRecordsNoRevision(t *testing.T) {
	root := t.TempDir()

	result, err := Run(root, "SP-202", "true")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.HeadRev != "" {
		t.Fatalf("HeadRev = %q, want an empty string", result.HeadRev)
	}
	if result.TreeClean != nil {
		t.Fatalf("TreeClean = %v, want nil", result.TreeClean)
	}
}

func initVerifyTestRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	runFixtureGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	runFixtureGit("init")
	runFixtureGit("config", "user.email", "test@example.com")
	runFixtureGit("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runFixtureGit("add", ".")
	runFixtureGit("commit", "-m", "initial")
	return root
}
