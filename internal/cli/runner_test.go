package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
)

func setupConfigScopes(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	setupShipProofDir(t, root)
	configHome := filepath.Join(t.TempDir(), "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
	return root, configHome
}

// P3: configuration reads and writes at local and global scope.
func TestConfigSetGlobalThenGet(t *testing.T) {
	_, configHome := setupConfigScopes(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"config", "set", "agent.runner", "codex", "--global"}, stdout, stderr); code != 0 {
		t.Fatalf("set exit = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(configHome, "shipproof", "config.yaml")); err != nil {
		t.Fatalf("global config not written: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"config", "get", "agent.runner"}, stdout, stderr); code != 0 {
		t.Fatalf("get exit = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "codex" {
		t.Fatalf("get output = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "global") {
		t.Fatalf("scope not reported: %q", stderr.String())
	}
}

func TestConfigSetLocalOverridesGlobal(t *testing.T) {
	setupConfigScopes(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if code := Run([]string{"config", "set", "agent.runner", "codex", "--global"}, stdout, stderr); code != 0 {
		t.Fatalf("set global exit = %d", code)
	}
	if code := Run([]string{"config", "set", "agent.runner", "opencode", "--local"}, stdout, stderr); code != 0 {
		t.Fatalf("set local exit = %d", code)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"config", "get", "agent.runner"}, stdout, stderr); code != 0 {
		t.Fatalf("get exit = %d", code)
	}
	if strings.TrimSpace(stdout.String()) != "opencode" {
		t.Fatalf("get output = %q", stdout.String())
	}
}

func TestConfigRejectsCredentialKey(t *testing.T) {
	setupConfigScopes(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"config", "set", "agent.api_key", "sk-test", "--local"}, stdout, stderr); code == 0 {
		t.Fatal("ShipProof must refuse to store a credential")
	}
}

func TestConfigGetMissingKey(t *testing.T) {
	setupConfigScopes(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"config", "get", "agent.runner"}, stdout, stderr); code != 1 {
		t.Fatalf("exit = %d", code)
	}
}

// stubProbeRunner reports a fixed probe result.
type stubProbeRunner struct{ status agent.RunnerStatus }

func (runner stubProbeRunner) Probe(ctx context.Context) (agent.RunnerStatus, error) {
	return runner.status, nil
}

func (runner stubProbeRunner) Run(ctx context.Context, req agent.RunRequest) (agent.RunResult, error) {
	return agent.RunResult{}, nil
}

func withStubRegistry(t *testing.T, statuses map[string]agent.RunnerStatus) {
	t.Helper()
	previous := defaultRegistry
	registry := agent.NewRegistry()
	for name, status := range statuses {
		fixed := status
		registry.MustRegister(name, func(config agent.RunnerConfig) (agent.AgentRunner, error) {
			return stubProbeRunner{status: fixed}, nil
		})
	}
	defaultRegistry = registry
	t.Cleanup(func() { defaultRegistry = previous })
}

func TestRunnerListReportsStatus(t *testing.T) {
	setupConfigScopes(t)
	withStubRegistry(t, map[string]agent.RunnerStatus{
		"codex":  {Installed: true, Authenticated: true, Version: "0.9.1"},
		"claude": {Installed: true, Version: "2.0.0"},
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"runner", "list"}, stdout, stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "codex\tinstalled=true\tauthenticated=true\tversion=0.9.1") {
		t.Fatalf("output = %s", output)
	}
	if !strings.Contains(output, "claude\tinstalled=true\tauthenticated=false") {
		t.Fatalf("output = %s", output)
	}
}

// P4: `runner doctor` reports status and never prints a credential.
func TestRunnerDoctorReportsStatusAndNeverPrintsCredential(t *testing.T) {
	setupConfigScopes(t)
	const secret = "sk-ant-super-secret-value"
	t.Setenv("ANTHROPIC_API_KEY", secret)
	t.Setenv("SHIPPROOF_RUNNER", "")

	withStubRegistry(t, map[string]agent.RunnerStatus{
		"codex": {Installed: true, Authenticated: true, Version: "0.9.1", Detail: "Codex CLI is ready."},
		"claude": {
			Installed: true,
			Detail:    "Claude Code CLI is not authenticated. Run `claude` and complete login.",
		},
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"runner", "doctor"}, stdout, stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	output := stdout.String() + stderr.String()
	if strings.Contains(output, secret) {
		t.Fatal("doctor printed a credential")
	}
	for _, want := range []string{"codex: ready", "claude: not usable", "selected runner: codex", "capabilities:"} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output)
		}
	}
	if !strings.Contains(output, "does not store provider credentials") {
		t.Fatalf("doctor output = %s", output)
	}
}

func TestRunnerDoctorReportsAmbiguousSelection(t *testing.T) {
	setupConfigScopes(t)
	t.Setenv("SHIPPROOF_RUNNER", "")
	withStubRegistry(t, map[string]agent.RunnerStatus{
		"codex":  {Installed: true, Authenticated: true},
		"claude": {Installed: true, Authenticated: true},
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"runner", "doctor"}, stdout, stderr); code != 1 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "selected runner: none") {
		t.Fatalf("output = %s", stdout.String())
	}
}

func TestRunnerDoctorUsesEnvironmentOverride(t *testing.T) {
	setupConfigScopes(t)
	t.Setenv("SHIPPROOF_RUNNER", "claude")
	withStubRegistry(t, map[string]agent.RunnerStatus{
		"codex":  {Installed: true, Authenticated: true},
		"claude": {Installed: true, Authenticated: true},
	})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"runner", "doctor"}, stdout, stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "selected runner: claude (from environment)") {
		t.Fatalf("output = %s", stdout.String())
	}
}
