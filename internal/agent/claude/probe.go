// Package claude adapts the Claude Code CLI to the ShipProof AgentRunner
// interface. Adding this adapter needs no change to AgentRunner.
package claude

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alternayte/shipproof/internal/agent"
)

// Name is the registry name of this runner.
const Name = "claude"

const defaultBinary = "claude"

// Runner executes work through the Claude Code CLI over a subprocess
// transport.
type Runner struct {
	binary string
	model  string
}

// New builds a Claude runner from its configuration. The `path` setting
// overrides the executable. The configuration must not hold a credential.
func New(config agent.RunnerConfig) (agent.AgentRunner, error) {
	binary := strings.TrimSpace(config.Setting("path"))
	if binary == "" {
		binary = defaultBinary
	}
	return Runner{binary: binary, model: strings.TrimSpace(config.Setting("model"))}, nil
}

func capabilities() agent.RunnerCapabilities {
	return agent.RunnerCapabilities{
		Resume:           true,
		ReadOnly:         true,
		WorkspaceWrite:   true,
		StructuredOutput: true,
		Streaming:        true,
	}
}

// Probe reports installation, authentication, version, and capabilities.
// It reports whether a credential exists. It never reads or prints one.
func (runner Runner) Probe(ctx context.Context) (agent.RunnerStatus, error) {
	status := agent.RunnerStatus{Capabilities: capabilities()}

	path, err := exec.LookPath(runner.binary)
	if err != nil {
		status.Detail = "Claude Code CLI not found. Install it, then run `claude`."
		return status, nil
	}
	status.Installed = true

	version, _, err := runCapture(ctx, path, "", "--version")
	if err != nil {
		status.Detail = "Claude Code CLI did not report a version."
		return status, nil
	}
	status.Version = firstLine(version)

	if !credentialPresent() {
		status.Detail = "Claude Code CLI is not authenticated. Run `claude` and complete login."
		return status, nil
	}
	status.Authenticated = true
	status.Detail = "Claude Code CLI is ready."
	return status, nil
}

// credentialPresent reports the presence of a Claude Code credential. It never
// returns credential content.
func credentialPresent() bool {
	for _, name := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN"} {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return true
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", ".credentials.json")); err == nil {
		return true
	}
	return false
}

func runCapture(ctx context.Context, binary, dir string, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func firstLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
