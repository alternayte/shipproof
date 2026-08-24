package codex

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
)

var _ agent.AgentRunner = Runner{}

// stubBinary writes an executable stand-in for the Codex CLI.
func stubBinary(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stub process needs a POSIX shell")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "codex")
	body := "#!/bin/sh\n" + script
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

const readyStub = `
case "$1" in
  --version) echo "codex-cli 0.9.1"; exit 0 ;;
  login) [ "$2" = "status" ] && exit 0 ;;
  exec)
    shift
    echo "args: $*" > "$STUB_LOG"
    pwd -P >> "$STUB_LOG"
    echo "codex finished"
    exit 0 ;;
esac
exit 1
`

func newRunner(t *testing.T, path string, settings map[string]string) agent.AgentRunner {
	t.Helper()
	if settings == nil {
		settings = map[string]string{}
	}
	settings["path"] = path
	runner, err := New(agent.RunnerConfig{Name: Name, Settings: settings})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	return runner
}

// P5: the runner probes.
func TestProbeReportsReadyRunner(t *testing.T) {
	runner := newRunner(t, stubBinary(t, readyStub), nil)
	status, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !status.Installed || !status.Authenticated {
		t.Fatalf("status = %+v", status)
	}
	if status.Version != "codex-cli 0.9.1" {
		t.Fatalf("version = %q", status.Version)
	}
	if !status.Capabilities.WorkspaceWrite || !status.Capabilities.ReadOnly {
		t.Fatalf("capabilities = %+v", status.Capabilities)
	}
}

func TestProbeReportsMissingBinary(t *testing.T) {
	runner := newRunner(t, filepath.Join(t.TempDir(), "absent-codex"), nil)
	status, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if status.Installed || status.Usable() {
		t.Fatalf("status = %+v", status)
	}
	if !strings.Contains(status.Detail, "Install") {
		t.Fatalf("detail = %q", status.Detail)
	}
}

func TestProbeReportsUnauthenticated(t *testing.T) {
	stub := stubBinary(t, `
case "$1" in
  --version) echo "codex-cli 0.9.1"; exit 0 ;;
esac
exit 1
`)
	runner := newRunner(t, stub, nil)
	status, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !status.Installed || status.Authenticated {
		t.Fatalf("status = %+v", status)
	}
	if !strings.Contains(status.Detail, "codex login") {
		t.Fatalf("detail = %q", status.Detail)
	}
}

// P5: the runner runs.
func TestRunImplementerUsesWorkspaceWriteSandbox(t *testing.T) {
	stub := stubBinary(t, readyStub)
	workspace := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "stub.log")
	t.Setenv("STUB_LOG", logPath)

	runner := newRunner(t, stub, map[string]string{"model": "o4-mini"})
	result, err := runner.Run(context.Background(), agent.RunRequest{
		Workspace:    workspace,
		Change:       agent.Change{ID: "CH-001"},
		Role:         agent.RoleImplementer,
		Instructions: "Implement the approved change.",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != agent.RunStatusSuccess {
		t.Fatalf("status = %q", result.Status)
	}
	if result.Summary != "codex finished" {
		t.Fatalf("summary = %q", result.Summary)
	}
	if result.Metadata["sandbox"] != "workspace-write" || result.Metadata["model"] != "o4-mini" {
		t.Fatalf("metadata = %+v", result.Metadata)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read stub log: %v", err)
	}
	log := string(data)
	if !strings.Contains(log, "--sandbox workspace-write") {
		t.Fatalf("stub log = %q", log)
	}
	if !strings.Contains(log, "--model o4-mini") {
		t.Fatalf("stub log = %q", log)
	}
	if !strings.Contains(log, "CH-001") {
		t.Fatalf("prompt did not reach the runner: %q", log)
	}
	resolvedWorkspace, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatalf("resolve workspace: %v", err)
	}
	if !strings.Contains(log, resolvedWorkspace) {
		t.Fatalf("runner did not execute in the workspace: %q", log)
	}
}

func TestRunReviewerUsesReadOnlySandbox(t *testing.T) {
	stub := stubBinary(t, readyStub)
	logPath := filepath.Join(t.TempDir(), "stub.log")
	t.Setenv("STUB_LOG", logPath)

	runner := newRunner(t, stub, nil)
	if _, err := runner.Run(context.Background(), agent.RunRequest{
		Workspace:    t.TempDir(),
		Change:       agent.Change{ID: "CH-002"},
		Role:         agent.RoleReviewer,
		Instructions: "Review the change.",
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, _ := os.ReadFile(logPath)
	if !strings.Contains(string(data), "--sandbox read-only") {
		t.Fatalf("stub log = %q", string(data))
	}
}

func TestRunReportsFailedStatusOnNonZeroExit(t *testing.T) {
	stub := stubBinary(t, `
case "$1" in
  exec) echo "codex could not finish"; exit 3 ;;
esac
exit 1
`)
	runner := newRunner(t, stub, nil)
	result, err := runner.Run(context.Background(), agent.RunRequest{
		Workspace:    t.TempDir(),
		Change:       agent.Change{ID: "CH-003"},
		Role:         agent.RoleImplementer,
		Instructions: "Implement the approved change.",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != agent.RunStatusFailed {
		t.Fatalf("status = %q", result.Status)
	}
	if result.Metadata["exit_code"] != "3" {
		t.Fatalf("metadata = %+v", result.Metadata)
	}
}
