// Package codex adapts the Codex CLI to the ShipProof AgentRunner interface.
// Every process detail stays inside this package.
package codex

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/alternayte/shipproof/internal/agent"
)

// Name is the registry name of this runner.
const Name = "codex"

const defaultBinary = "codex"

// Runner executes work through the Codex CLI over a subprocess transport.
type Runner struct {
	binary string
	model  string
}

// New builds a Codex runner from its configuration. The `path` setting
// overrides the executable. The configuration must not hold a credential.
func New(config agent.RunnerConfig) (agent.AgentRunner, error) {
	binary := strings.TrimSpace(config.Setting("path"))
	if binary == "" {
		binary = defaultBinary
	}
	return Runner{binary: binary, model: strings.TrimSpace(config.Setting("model"))}, nil
}

// capabilities describes what the Codex CLI supports in v0.
func capabilities() agent.RunnerCapabilities {
	return agent.RunnerCapabilities{
		Resume:           false,
		ReadOnly:         true,
		WorkspaceWrite:   true,
		StructuredOutput: false,
		Streaming:        true,
	}
}

// Probe reports installation, authentication, version, and capabilities.
// It never reports credential content.
func (runner Runner) Probe(ctx context.Context) (agent.RunnerStatus, error) {
	status := agent.RunnerStatus{Capabilities: capabilities()}

	path, err := exec.LookPath(runner.binary)
	if err != nil {
		status.Detail = "Codex CLI not found. Install it, then run `codex login`."
		return status, nil
	}
	status.Installed = true

	version, _, err := runCapture(ctx, path, "", "--version")
	if err != nil {
		status.Detail = "Codex CLI did not report a version."
		return status, nil
	}
	status.Version = firstLine(version)

	if _, _, err := runCapture(ctx, path, "", "login", "status"); err != nil {
		status.Detail = "Codex CLI is not authenticated. Run `codex login`."
		return status, nil
	}
	status.Authenticated = true
	status.Detail = "Codex CLI is ready."
	return status, nil
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
