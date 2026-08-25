package verification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePlan(t *testing.T) {
	plan := New("SP-42")
	plan.Requirements = []Item{{ID: "R-1", Proof: []Proof{{Type: "integration-test", Target: "returns 202", Command: "go test ./..."}}}}
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsMissingProof(t *testing.T) {
	plan := New("SP-42")
	plan.Requirements = []Item{{ID: "R-1"}}
	if err := plan.Validate(); err == nil {
		t.Fatal("expected missing proof error")
	}
}

func TestInitializeCreatesEditableSkeleton(t *testing.T) {
	root := t.TempDir()
	path, err := Initialize(root, "SP-42")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.ChangeID != "SP-42" {
		t.Fatalf("change id = %q", plan.ChangeID)
	}
}

func TestProofRejectsNeitherCommandNorHuman(t *testing.T) {
	t.Parallel()

	plan := Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []Item{{
			ID:    "SP-021-R1",
			Proof: []Proof{{Type: "test", Target: "internal/verification"}},
		}},
	}
	err := plan.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want an error for a proof with no command and no human flag")
	}
	if !strings.Contains(err.Error(), "command") || !strings.Contains(err.Error(), "human") {
		t.Fatalf("Validate() error = %v, want it to name both forms", err)
	}
}

func TestProofAcceptsHumanWithRationale(t *testing.T) {
	t.Parallel()

	plan := Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []Item{{
			ID: "SP-021-R1",
			Proof: []Proof{{
				Type:      "human",
				Target:    "skills/plan-verification/SKILL.md",
				Human:     true,
				Rationale: "A person reads the skill and confirms it.",
			}},
		}},
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProofRejectsHumanWithoutRationale(t *testing.T) {
	t.Parallel()

	plan := Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []Item{{
			ID:    "SP-021-R1",
			Proof: []Proof{{Type: "human", Target: "x", Human: true}},
		}},
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error for a human proof with no rationale")
	}
}

func TestProofRejectsAcceptanceOnAnAutomatedProof(t *testing.T) {
	t.Parallel()

	plan := Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []Item{{
			ID: "SP-021-R1",
			Proof: []Proof{{
				Type:       "test",
				Target:     "x",
				Command:    "go test ./...",
				AcceptedAt: "2026-08-25T10:00:00Z",
			}},
		}},
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error; a command proof cannot carry an acceptance")
	}
}

func TestProofRejectsAMalformedAcceptanceTimestamp(t *testing.T) {
	t.Parallel()

	plan := Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []Item{{
			ID: "SP-021-R1",
			Proof: []Proof{{
				Type:       "human",
				Target:     "x",
				Human:      true,
				Rationale:  "A person confirms it.",
				AcceptedAt: "yesterday",
			}},
		}},
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error for a malformed acceptance timestamp")
	}
}

func TestEveryRecordedPlanStillLoads(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("..", "..", ".shipproof", "changes", "*", "verification.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Skip("no recorded verification plan in this workspace")
	}
	for _, path := range paths {
		if _, err := Load(path); err != nil {
			t.Fatalf("Load(%s) error = %v", path, err)
		}
	}
}

func TestProofRejectsBothCommandAndHuman(t *testing.T) {
	t.Parallel()

	plan := Plan{
		SchemaVersion: "0.1",
		ChangeID:      "SP-021",
		Requirements: []Item{{
			ID: "SP-021-R1",
			Proof: []Proof{{
				Type:      "test",
				Target:    "x",
				Command:   "go test ./...",
				Human:     true,
				Rationale: "A person reads it.",
			}},
		}},
	}
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error; a proof is one form or the other")
	}
}
