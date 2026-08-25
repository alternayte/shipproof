package verification

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Proof names one way to prove a requirement. A proof carries either a
// command that a machine runs, or a human flag with a rationale. Nothing else
// separates a human check from an incomplete one.
type Proof struct {
	Type    string `json:"type"`
	Target  string `json:"target"`
	Command string `json:"command,omitempty"`
	// Human marks a proof that only a person can perform.
	Human bool `json:"human,omitempty"`
	// Rationale states why no machine can perform this proof.
	Rationale string `json:"rationale,omitempty"`
	// AcceptedAt records when a person accepted a human proof, in RFC 3339.
	// It is empty until that person accepts it.
	AcceptedAt string `json:"accepted_at,omitempty"`
}

// IsHuman reports whether only a person can perform this proof.
func (proof Proof) IsHuman() bool { return proof.Human }

// IsAutomated reports whether this proof carries a command a machine runs.
func (proof Proof) IsAutomated() bool {
	return !proof.Human && strings.TrimSpace(proof.Command) != ""
}

// Accepted reports whether a person recorded acceptance of a human proof.
func (proof Proof) Accepted() bool {
	return proof.Human && strings.TrimSpace(proof.AcceptedAt) != ""
}

type Item struct {
	ID        string  `json:"id"`
	Statement string  `json:"statement,omitempty"`
	Proof     []Proof `json:"proof"`
}

type Plan struct {
	SchemaVersion string `json:"schema_version"`
	ChangeID      string `json:"change_id"`
	Requirements  []Item `json:"requirements"`
	Invariants    []Item `json:"invariants"`
}

func New(changeID string) Plan {
	return Plan{
		SchemaVersion: "0.1",
		ChangeID:      strings.TrimSpace(changeID),
		Requirements:  []Item{},
		Invariants:    []Item{},
	}
}

func (plan Plan) Validate() error {
	if plan.SchemaVersion != "0.1" {
		return fmt.Errorf("schema_version must be %q", "0.1")
	}
	if strings.TrimSpace(plan.ChangeID) == "" {
		return errors.New("change_id is required")
	}
	seen := map[string]struct{}{}
	for _, group := range []struct {
		name  string
		items []Item
	}{{"requirements", plan.Requirements}, {"invariants", plan.Invariants}} {
		for _, item := range group.items {
			if strings.TrimSpace(item.ID) == "" {
				return fmt.Errorf("%s item id is required", group.name)
			}
			if _, exists := seen[item.ID]; exists {
				return fmt.Errorf("duplicate verification item id %q", item.ID)
			}
			seen[item.ID] = struct{}{}
			if len(item.Proof) == 0 {
				return fmt.Errorf("verification item %q requires at least one proof", item.ID)
			}
			for index, proof := range item.Proof {
				if strings.TrimSpace(proof.Type) == "" {
					return fmt.Errorf("verification item %q proof %d type is required", item.ID, index+1)
				}
				if strings.TrimSpace(proof.Target) == "" {
					return fmt.Errorf("verification item %q proof %d target is required", item.ID, index+1)
				}
				if err := validateProofForm(item.ID, index, proof); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func Path(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "changes", changeID, "verification.json")
}

func Initialize(root, changeID string) (string, error) {
	plan := New(changeID)
	path := Path(root, changeID)
	if _, err := os.Stat(path); err == nil {
		return path, fmt.Errorf("verification plan already exists for %q", changeID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return path, fmt.Errorf("inspect verification plan: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, fmt.Errorf("create change directory: %w", err)
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return path, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return path, fmt.Errorf("write verification plan: %w", err)
	}
	return path, nil
}

func Load(path string) (Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, fmt.Errorf("read verification plan: %w", err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return Plan{}, fmt.Errorf("parse verification plan: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// validateProofForm enforces the two proof forms. A proof with neither a
// command nor a human flag is an incomplete proof, and the plan must not hide
// it behind a passing check.
func validateProofForm(itemID string, index int, proof Proof) error {
	hasCommand := strings.TrimSpace(proof.Command) != ""
	switch {
	case proof.Human && hasCommand:
		return fmt.Errorf("verification item %q proof %d carries both a command and human: true; a proof is one form or the other", itemID, index+1)
	case proof.Human:
		if strings.TrimSpace(proof.Rationale) == "" {
			return fmt.Errorf("verification item %q proof %d is human and needs a rationale", itemID, index+1)
		}
	case hasCommand:
		if strings.TrimSpace(proof.AcceptedAt) != "" {
			return fmt.Errorf("verification item %q proof %d carries a command and accepted_at; only a human proof records acceptance", itemID, index+1)
		}
	default:
		return fmt.Errorf("verification item %q proof %d needs a command or human: true with a rationale", itemID, index+1)
	}
	if accepted := strings.TrimSpace(proof.AcceptedAt); accepted != "" {
		if _, err := time.Parse(time.RFC3339, accepted); err != nil {
			return fmt.Errorf("verification item %q proof %d accepted_at is not an RFC 3339 timestamp: %v", itemID, index+1, err)
		}
	}
	return nil
}
