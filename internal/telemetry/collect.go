package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shipproof/shipproof/internal/agent"
	"github.com/shipproof/shipproof/internal/telemetry/claude"
	"github.com/shipproof/shipproof/internal/telemetry/opencode"
)

func Collect(root, changeID, adapterName, projectDir string) error {
	adapter, err := adapterByName(adapterName)
	if err != nil {
		return err
	}

	if projectDir == "" {
		projectDir = root
	}

	run, err := adapter.Collect(projectDir)
	if err != nil {
		return fmt.Errorf("collect telemetry from %s: %w", adapter.Name(), err)
	}

	run.Provider = adapter.Name()

	if run.StartedAt == "" {
		run.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if run.EndedAt == "" {
		run.EndedAt = time.Now().UTC().Format(time.RFC3339)
	}

	dir := filepath.Join(root, ".shipproof", "runs", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create runs directory: %w", err)
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("encode agent run record: %w", err)
	}
	data = append(data, '\n')

	path := filepath.Join(dir, "agent-run.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write agent run record: %w", err)
	}

	return nil
}

func adapterByName(name string) (agent.Adapter, error) {
	switch name {
	case "claude":
		return claude.NewAdapter(), nil
	case "opencode":
		return opencode.NewAdapter(), nil
	default:
		return nil, fmt.Errorf("unsupported adapter %q; use claude or opencode", name)
	}
}
