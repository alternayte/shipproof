package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shipproof/shipproof/internal/agent"
	"github.com/shipproof/shipproof/internal/repository"
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

	if err := applyCaptureLevel(root, dir, adapter, projectDir, &run); err != nil {
		return err
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

// applyCaptureLevel enforces the evidence capture profile from SDD §18.
//
// metadata: store timing and metadata only. RawLogRef keeps the original
// transcript location as a reference; nothing is copied.
//
// redacted: copy the raw transcript into agent-raw/ with recognized secret
// shapes masked.
//
// full: copy the raw transcript into agent-raw/ unchanged.
func applyCaptureLevel(root, runDir string, adapter agent.Adapter, projectDir string, run *agent.AgentRun) error {
	cfg, err := repository.LoadEvidenceConfig(root)
	if err != nil {
		return err
	}

	provider, ok := adapter.(agent.RawLogProvider)
	if !ok {
		return nil
	}

	rawPath, err := provider.RawLogPath(projectDir)
	if err != nil {
		return nil
	}

	switch cfg.Evidence.Capture {
	case repository.CaptureMetadata:
		run.RawLogRef = rawPath
		return nil
	case repository.CaptureRedacted, repository.CaptureFull:
		dest, err := copyRaw(root, runDir, rawPath, cfg.Evidence.Capture == repository.CaptureRedacted)
		if err != nil {
			return err
		}
		run.RawLogRef = dest
		return nil
	default:
		return nil
	}
}

// copyRaw copies a raw transcript file or directory into
// .shipproof/runs/<change-id>/agent-raw/. It returns the destination path
// relative to the repository root.
func copyRaw(root, runDir, rawPath string, redact bool) (string, error) {
	info, err := os.Stat(rawPath)
	if err != nil {
		return "", fmt.Errorf("inspect raw log %q: %w", rawPath, err)
	}

	dest := filepath.Join(runDir, "agent-raw")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", fmt.Errorf("create agent-raw directory: %w", err)
	}

	// When the provider exposes a directory, copy only the newest 10
	// files to keep the captured artifact bounded.
	files := []string{rawPath}
	if info.IsDir() {
		files, err = newestFiles(rawPath, 10)
		if err != nil {
			return "", err
		}
	}

	for _, file := range files {
		rel, err := filepath.Rel(filepath.Dir(rawPath), file)
		if err != nil {
			rel = filepath.Base(file)
		}
		target := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("create target directory: %w", err)
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read raw log %q: %w", file, err)
		}
		if redact {
			content = Redact(content)
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return "", fmt.Errorf("write captured log %q: %w", target, err)
		}
	}

	relDest, err := filepath.Rel(root, dest)
	if err != nil {
		relDest = dest
	}
	return filepath.ToSlash(relDest), nil
}

// newestFiles returns up to limit regular files ordered by modification time,
// newest first.
func newestFiles(dir string, limit int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read raw log directory %q: %w", dir, err)
	}

	type named struct {
		path    string
		modTime int64
	}
	var candidates []named
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		candidates = append(candidates, named{filepath.Join(dir, entry.Name()), info.ModTime().UnixNano()})
	}

	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].modTime > candidates[i].modTime {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	result := make([]string, 0, limit)
	for _, candidate := range candidates {
		result = append(result, candidate.path)
		if len(result) == limit {
			break
		}
	}
	return result, nil
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
