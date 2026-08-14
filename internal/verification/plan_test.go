package verification

import (
	"encoding/json"
	"os"
	"testing"
)

func TestValidatePlan(t *testing.T) {
	plan := New("SP-42")
	plan.Requirements = []Item{{ID: "R-1", Proof: []Proof{{Type: "integration-test", Target: "returns 202"}}}}
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
