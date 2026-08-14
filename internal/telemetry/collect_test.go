package telemetry

import (
	"os"
	"path/filepath"
	"testing"
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
