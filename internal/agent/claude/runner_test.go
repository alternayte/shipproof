package claude

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
)

// P6: ClaudeRunner satisfies the unchanged AgentRunner interface.
var _ agent.AgentRunner = Runner{}

func TestAgentRunnerInterfaceIsUnchanged(t *testing.T) {
	runnerType := reflect.TypeOf((*agent.AgentRunner)(nil)).Elem()
	if runnerType.NumMethod() != 2 {
		t.Fatalf("AgentRunner must still expose 2 methods, got %d", runnerType.NumMethod())
	}
	if runnerType.Method(0).Name != "Probe" || runnerType.Method(1).Name != "Run" {
		t.Fatalf("AgentRunner methods changed: %s, %s", runnerType.Method(0).Name, runnerType.Method(1).Name)
	}
}

func stubBinary(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stub process needs a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

const readyStub = `
if [ "$1" = "--version" ]; then echo "2.0.0 (Claude Code)"; exit 0; fi
echo "args: $*" > "$STUB_LOG"
pwd -P >> "$STUB_LOG"
cat <<'JSON'
{"session_id":"sess-42","result":"Implemented the change.","is_error":false,"subtype":"success"}
JSON
exit 0
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

func TestProbeReportsReadyRunner(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-value")
	runner := newRunner(t, stubBinary(t, readyStub), nil)
	status, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !status.Usable() {
		t.Fatalf("status = %+v", status)
	}
	if status.Version != "2.0.0 (Claude Code)" {
		t.Fatalf("version = %q", status.Version)
	}
	if !status.Capabilities.Resume || !status.Capabilities.StructuredOutput {
		t.Fatalf("capabilities = %+v", status.Capabilities)
	}
}

func TestProbeNeverReportsCredentialContent(t *testing.T) {
	const secret = "sk-ant-secret-value"
	t.Setenv("ANTHROPIC_API_KEY", secret)
	runner := newRunner(t, stubBinary(t, readyStub), nil)
	status, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if strings.Contains(status.Detail, secret) || strings.Contains(status.Version, secret) {
		t.Fatalf("probe leaked a credential: %+v", status)
	}
}

func TestProbeReportsUnauthenticated(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	t.Setenv("HOME", t.TempDir())
	runner := newRunner(t, stubBinary(t, readyStub), nil)
	status, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !status.Installed || status.Authenticated {
		t.Fatalf("status = %+v", status)
	}
}

func TestRunParsesStructuredOutput(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "stub.log")
	t.Setenv("STUB_LOG", logPath)
	runner := newRunner(t, stubBinary(t, readyStub), nil)

	result, err := runner.Run(context.Background(), agent.RunRequest{
		Workspace:    t.TempDir(),
		Change:       agent.Change{ID: "CH-010"},
		Role:         agent.RoleImplementer,
		Instructions: "Implement the approved change.",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != agent.RunStatusSuccess {
		t.Fatalf("status = %q", result.Status)
	}
	if result.SessionRef != "sess-42" {
		t.Fatalf("session ref = %q", result.SessionRef)
	}
	if result.Summary != "Implemented the change." {
		t.Fatalf("summary = %q", result.Summary)
	}
	data, _ := os.ReadFile(logPath)
	if !strings.Contains(string(data), "--permission-mode acceptEdits") {
		t.Fatalf("stub log = %q", string(data))
	}
}

func TestRunReviewerUsesPlanPermissionMode(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "stub.log")
	t.Setenv("STUB_LOG", logPath)
	runner := newRunner(t, stubBinary(t, readyStub), nil)
	if _, err := runner.Run(context.Background(), agent.RunRequest{
		Workspace:    t.TempDir(),
		Change:       agent.Change{ID: "CH-011"},
		Role:         agent.RoleReviewer,
		Instructions: "Review the change.",
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	data, _ := os.ReadFile(logPath)
	if !strings.Contains(string(data), "--permission-mode plan") {
		t.Fatalf("stub log = %q", string(data))
	}
}

func TestRunReportsErrorResult(t *testing.T) {
	stub := stubBinary(t, `
if [ "$1" = "--version" ]; then echo "2.0.0"; exit 0; fi
echo '{"session_id":"sess-9","result":"blocked","is_error":true,"subtype":"error_during_execution"}'
exit 0
`)
	runner := newRunner(t, stub, nil)
	result, err := runner.Run(context.Background(), agent.RunRequest{
		Workspace:    t.TempDir(),
		Change:       agent.Change{ID: "CH-012"},
		Role:         agent.RoleImplementer,
		Instructions: "Implement the approved change.",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != agent.RunStatusFailed {
		t.Fatalf("status = %q", result.Status)
	}
}
