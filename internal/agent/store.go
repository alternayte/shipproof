package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ExecutionPath returns the durable execution record path for one change.
func ExecutionPath(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "changes", changeID, "execution.json")
}

// SaveExecution writes the durable execution record.
func SaveExecution(root string, record ExecutionRecord) (string, error) {
	path := ExecutionPath(root, record.Change)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create change directory: %w", err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode execution record: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write execution record: %w", err)
	}
	return path, nil
}

// LoadExecution reads the durable execution record.
func LoadExecution(root, changeID string) (ExecutionRecord, error) {
	var record ExecutionRecord
	data, err := os.ReadFile(ExecutionPath(root, changeID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return record, err
		}
		return record, fmt.Errorf("read execution record: %w", err)
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return record, fmt.Errorf("parse execution record: %w", err)
	}
	return record, nil
}
