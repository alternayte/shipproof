package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/alternayte/shipproof/internal/agent"
	"github.com/alternayte/shipproof/internal/agent/claude"
	"github.com/alternayte/shipproof/internal/agent/codex"
	"github.com/alternayte/shipproof/internal/agent/opencode"
	"github.com/alternayte/shipproof/internal/repository"
)

// defaultRegistry holds the v0 adapters. Test code can replace it.
var defaultRegistry = newDefaultRegistry()

func newDefaultRegistry() *agent.Registry {
	registry := agent.NewRegistry()
	registry.MustRegister(codex.Name, codex.New)
	registry.MustRegister(claude.Name, claude.New)
	registry.MustRegister(opencode.Name, opencode.New)
	return registry
}

// probedRunner pairs one runner name with its probe result.
type probedRunner struct {
	Name   string
	Status agent.RunnerStatus
	Err    error
}

// probeTimeout bounds one probe. A probe must never block the command line.
const probeTimeout = 15 * time.Second

func probeAll(ctx context.Context, registry *agent.Registry, config repository.AgentConfig) []probedRunner {
	names := registry.Names()
	sort.Strings(names)
	results := make([]probedRunner, 0, len(names))
	for _, name := range names {
		result := probedRunner{Name: name}
		runner, err := registry.Build(agent.RunnerConfig{Name: name, Settings: config.Runners[name]})
		if err != nil {
			result.Err = err
			results = append(results, result)
			continue
		}
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		status, err := runner.Probe(probeCtx)
		cancel()
		result.Status = status
		result.Err = err
		results = append(results, result)
	}
	return results
}

func usableNames(results []probedRunner) []string {
	var names []string
	for _, result := range results {
		if result.Err == nil && result.Status.Usable() {
			names = append(names, result.Name)
		}
	}
	return names
}

// runRunner implements `shipproof runner list|doctor`.
func runRunner(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof runner <list|doctor>")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		root = "."
	}
	config, err := repository.LoadAgentConfig(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	results := probeAll(context.Background(), defaultRegistry, config)

	switch args[0] {
	case "list":
		for _, result := range results {
			fmt.Fprintf(stdout, "%s\tinstalled=%t\tauthenticated=%t\tversion=%s\n",
				result.Name, result.Status.Installed, result.Status.Authenticated, versionOrUnknown(result.Status.Version))
		}
		return 0
	case "doctor":
		local, global, err := scopedAgentConfig(root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return doctor(stdout, results, local, global)
	default:
		fmt.Fprintf(stderr, "unknown runner command %q\n", args[0])
		return 2
	}
}

// doctor prints actionable setup instructions. It never prints a credential.
func doctor(stdout io.Writer, results []probedRunner, local, global repository.AgentConfig) int {
	for _, result := range results {
		state := "not usable"
		if result.Err == nil && result.Status.Usable() {
			state = "ready"
		}
		fmt.Fprintf(stdout, "%s: %s\n", result.Name, state)
		if result.Err != nil {
			fmt.Fprintf(stdout, "  detail: probe failed\n")
			continue
		}
		fmt.Fprintf(stdout, "  installed: %t\n", result.Status.Installed)
		fmt.Fprintf(stdout, "  authenticated: %t\n", result.Status.Authenticated)
		fmt.Fprintf(stdout, "  version: %s\n", versionOrUnknown(result.Status.Version))
		fmt.Fprintf(stdout, "  capabilities: resume=%t read_only=%t workspace_write=%t structured_output=%t streaming=%t\n",
			result.Status.Capabilities.Resume, result.Status.Capabilities.ReadOnly, result.Status.Capabilities.WorkspaceWrite,
			result.Status.Capabilities.StructuredOutput, result.Status.Capabilities.Streaming)
		if result.Status.Detail != "" {
			fmt.Fprintf(stdout, "  detail: %s\n", result.Status.Detail)
		}
	}

	selection, err := agent.Resolve(resolveInput("", usableNames(results), local, global))
	if err != nil {
		fmt.Fprintf(stdout, "\nselected runner: none — %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "\nselected runner: %s (from %s)\n", selection.Name, selection.Source)
	for _, result := range results {
		if result.Name == selection.Name && !result.Status.Usable() {
			fmt.Fprintf(stdout, "warning: the selected runner is not usable. `shipproof run` returns BLOCKED.\n")
		}
	}
	fmt.Fprintln(stdout, "ShipProof does not store provider credentials. Authenticate with the runner itself.")
	return 0
}

func versionOrUnknown(version string) string {
	if version == "" {
		return "unknown"
	}
	return version
}

// scopedAgentConfig reads the repository and the user configuration separately
// so that runner resolution can report the precedence level that decided.
func scopedAgentConfig(root string) (repository.AgentConfig, repository.AgentConfig, error) {
	local, err := repository.LoadAgentConfigScope(root, repository.ScopeLocal)
	if err != nil {
		return local, repository.AgentConfig{}, err
	}
	global, err := repository.LoadAgentConfigScope(root, repository.ScopeGlobal)
	if err != nil {
		return local, global, err
	}
	return local, global, nil
}

// resolveInput builds the precedence input for runner resolution.
func resolveInput(cliOverride string, usable []string, local, global repository.AgentConfig) agent.ResolveInput {
	return agent.ResolveInput{
		CLIOverride: cliOverride,
		Environment: os.Getenv(agent.EnvRunner),
		RepoConfig:  local.Runner,
		UserConfig:  global.Runner,
		Usable:      usable,
	}
}
