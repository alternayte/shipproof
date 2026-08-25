// Package requirements holds the ShipProof requirement set. The back half of
// the product needs one thing from a source document: a set of requirements
// with stable identifiers. The tie check, the coverage matrix, and the
// unexplained-change report all rest on this sidecar.
//
// No adopter edits a source document. Every adopter writes a sidecar beside
// the change record.
package requirements

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

// Provenance names how ShipProof learned a requirement. No adopter may write
// inferred. A requirement that a model guessed is not a requirement.
type Provenance string

const (
	// Observed marks a requirement that a parser read from a document with a
	// known format. No model took part.
	Observed Provenance = "observed"
	// Human marks a requirement that a person confirmed.
	Human Provenance = "human"
)

// AdopterNative names the adopter that parses a docs/changes/ document.
const AdopterNative = "native"

// AdopterForeign names the adopter that proposes an extraction from any other
// document and waits for a human confirmation.
const AdopterForeign = "foreign"

type Requirement struct {
	ID           string     `json:"id"`
	Statement    string     `json:"statement"`
	SourceAnchor string     `json:"source_anchor,omitempty"`
	Provenance   Provenance `json:"provenance"`
	// ConfirmedAt records when a person confirmed a foreign extraction. It is
	// empty for the native adopter, which needs no human step.
	ConfirmedAt string `json:"confirmed_at,omitempty"`
}

type Set struct {
	SchemaVersion string        `json:"schema_version"`
	ChangeID      string        `json:"change_id"`
	Adopter       string        `json:"adopter"`
	SourcePath    string        `json:"source_path,omitempty"`
	Requirements  []Requirement `json:"requirements"`
}

// Path returns the sidecar location for one change.
func Path(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "changes", changeID, "requirements.json")
}

// Validate reports the first rule the set breaks.
func (set Set) Validate() error {
	if set.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %q", SchemaVersion)
	}
	if strings.TrimSpace(set.ChangeID) == "" {
		return errors.New("change_id is required")
	}
	switch set.Adopter {
	case AdopterNative, AdopterForeign:
	default:
		return fmt.Errorf("adopter must be %q or %q, not %q", AdopterNative, AdopterForeign, set.Adopter)
	}
	if len(set.Requirements) == 0 {
		return errors.New("the requirement set holds no requirement")
	}
	seen := map[string]struct{}{}
	for index, requirement := range set.Requirements {
		if strings.TrimSpace(requirement.ID) == "" {
			return fmt.Errorf("requirement %d has no id", index+1)
		}
		if _, exists := seen[requirement.ID]; exists {
			return fmt.Errorf("duplicate requirement id %q", requirement.ID)
		}
		seen[requirement.ID] = struct{}{}
		if strings.TrimSpace(requirement.Statement) == "" {
			return fmt.Errorf("requirement %q has no statement", requirement.ID)
		}
		switch requirement.Provenance {
		case Observed, Human:
		default:
			return fmt.Errorf("requirement %q carries provenance %q; only %q and %q are allowed",
				requirement.ID, requirement.Provenance, Observed, Human)
		}
	}
	return nil
}

// Save validates the set and writes the sidecar.
func Save(root string, set Set) (string, error) {
	if err := set.Validate(); err != nil {
		return "", err
	}
	path := Path(root, set.ChangeID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create change directory: %w", err)
	}
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write requirement set: %w", err)
	}
	return path, nil
}

// Load reads and validates the sidecar for one change.
func Load(root, changeID string) (Set, error) {
	path := Path(root, changeID)
	data, err := os.ReadFile(path)
	if err != nil {
		return Set{}, fmt.Errorf("read requirement set: %w", err)
	}
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return Set{}, fmt.Errorf("parse requirement set: %w", err)
	}
	if err := set.Validate(); err != nil {
		return Set{}, err
	}
	return set, nil
}

// Exists reports whether a sidecar is present. A caller that must tolerate an
// absent sidecar uses this instead of reading the error of Load.
func Exists(root, changeID string) bool {
	_, err := os.Stat(Path(root, changeID))
	return err == nil
}
