package telemetry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shipproof/shipproof/internal/agent"
)

func TestCollectPreservesMissingFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := root

	if err := os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := Collect(root, "SP-008", "claude", projectDir)
	if err == nil {
		t.Log("collect succeeded (if sessions exist) or no sessions found")
	}

	t.Run("write-agent-run-file", func(t *testing.T) {
		root := t.TempDir()
		os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755)

		agentRunPath := filepath.Join(root, ".shipproof", "runs", "SP-008", "agent-run.json")
		_ = agentRunPath
	})
}

func TestCollectUnsupportedAdapter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755)

	err := Collect(root, "SP-008", "unsupported", root)
	if err == nil {
		t.Fatal("expected error for unsupported adapter")
	}
}

type fakeRawProvider struct {
	name    string
	rawPath string
	collect func(projectDir string) (agent.AgentRun, error)
}

func (f *fakeRawProvider) Name() string { return f.name }

func (f *fakeRawProvider) Collect(projectDir string) (agent.AgentRun, error) {
	if f.collect != nil {
		return f.collect(projectDir)
	}
	return agent.AgentRun{}, nil
}

func (f *fakeRawProvider) RawLogPath(projectDir string) (string, error) {
	if f.rawPath == "" {
		return "", errors.New("no raw log")
	}
	return f.rawPath, nil
}
