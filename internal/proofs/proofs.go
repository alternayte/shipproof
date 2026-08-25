// Package proofs holds the per-proof result artifact. The repository gate
// records one exit code for the whole verification contract. It cannot say
// which requirement a failure belongs to. This artifact answers that question,
// with one recorded result for each proof in the verification plan.
package proofs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SchemaVersion is the version that every ShipProof artifact carries.
const SchemaVersion = "0.1"

// Status names the outcome of one proof.
type Status string

const (
	// Pass marks an automated proof whose command exited zero.
	Pass Status = "pass"
	// Fail marks an automated proof whose command exited non-zero.
	Fail Status = "fail"
	// Human marks a proof that only a person can perform. ShipProof ran
	// nothing for it, and it claims nothing about it.
	Human Status = "human"
)

// Result records one proof.
type Result struct {
	RequirementID string `json:"requirement_id"`
	// ProofIndex is the zero-based position of the proof inside the plan item.
	ProofIndex int    `json:"proof_index"`
	Command    string `json:"command,omitempty"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Status     Status `json:"status"`
}

// Set is the whole artifact for one change at one revision.
type Set struct {
	SchemaVersion string   `json:"schema_version"`
	ChangeID      string   `json:"change_id"`
	HeadRev       string   `json:"head_rev,omitempty"`
	TreeClean     *bool    `json:"tree_clean,omitempty"`
	Timestamp     string   `json:"timestamp"`
	Results       []Result `json:"results"`
}

// Path returns the artifact location for one change.
func Path(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "runs", changeID, "proofs.json")
}

// Validate reports the first rule the set breaks.
func (set Set) Validate() error {
	if set.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %q", SchemaVersion)
	}
	if strings.TrimSpace(set.ChangeID) == "" {
		return errors.New("change_id is required")
	}
	if strings.TrimSpace(set.Timestamp) == "" {
		return errors.New("timestamp is required")
	}
	seen := map[string]struct{}{}
	for position, result := range set.Results {
		if strings.TrimSpace(result.RequirementID) == "" {
			return fmt.Errorf("result %d has no requirement_id", position+1)
		}
		if result.ProofIndex < 0 {
			return fmt.Errorf("result %d has a negative proof_index", position+1)
		}
		key := fmt.Sprintf("%s#%d", result.RequirementID, result.ProofIndex)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate result for %s proof %d", result.RequirementID, result.ProofIndex)
		}
		seen[key] = struct{}{}
		if result.DurationMs < 0 {
			return fmt.Errorf("result %s proof %d has a negative duration", result.RequirementID, result.ProofIndex)
		}
		if err := validateStatus(result); err != nil {
			return err
		}
	}
	return nil
}

// validateStatus keeps the status and the recorded numbers in agreement. A
// record that disagrees with itself is worse than no record.
func validateStatus(result Result) error {
	hasCommand := strings.TrimSpace(result.Command) != ""
	switch result.Status {
	case Human:
		if hasCommand {
			return fmt.Errorf("result %s proof %d is human and carries a command", result.RequirementID, result.ProofIndex)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("result %s proof %d is human and carries an exit code", result.RequirementID, result.ProofIndex)
		}
	case Pass:
		if !hasCommand {
			return fmt.Errorf("result %s proof %d passed and carries no command", result.RequirementID, result.ProofIndex)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("result %s proof %d is pass with exit code %d", result.RequirementID, result.ProofIndex, result.ExitCode)
		}
	case Fail:
		if !hasCommand {
			return fmt.Errorf("result %s proof %d failed and carries no command", result.RequirementID, result.ProofIndex)
		}
		if result.ExitCode == 0 {
			return fmt.Errorf("result %s proof %d is fail with exit code 0", result.RequirementID, result.ProofIndex)
		}
	default:
		return fmt.Errorf("result %s proof %d carries status %q; only %q, %q, and %q are allowed",
			result.RequirementID, result.ProofIndex, result.Status, Pass, Fail, Human)
	}
	return nil
}

// Save validates the set and writes the artifact.
func Save(root string, set Set) (string, error) {
	if err := set.Validate(); err != nil {
		return "", err
	}
	path := Path(root, set.ChangeID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create run directory: %w", err)
	}
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write proof results: %w", err)
	}
	return path, nil
}

// Load reads and validates the artifact for one change.
func Load(root, changeID string) (Set, error) {
	data, err := os.ReadFile(Path(root, changeID))
	if err != nil {
		return Set{}, fmt.Errorf("read proof results: %w", err)
	}
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return Set{}, fmt.Errorf("parse proof results: %w", err)
	}
	if err := set.Validate(); err != nil {
		return Set{}, err
	}
	return set, nil
}

// Exists reports whether the artifact is present. A caller that must tolerate
// an absent artifact uses this instead of reading the error of Load.
func Exists(root, changeID string) bool {
	_, err := os.Stat(Path(root, changeID))
	return err == nil
}
