package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
)

func sampleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "examples", "sample-project"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("the sample project does not exist: %v", err)
	}
	return root
}

// TestTheSampleProjectHoldsOneOpenChange holds requirement R1 of SP-038. The
// action names one change, so the sample must hold exactly that one.
func TestTheSampleProjectHoldsOneOpenChange(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(sampleRoot(t), ".shipproof", "changes"))
	if err != nil {
		t.Fatal(err)
	}
	var changes []string
	for _, entry := range entries {
		if entry.IsDir() {
			changes = append(changes, entry.Name())
		}
	}
	if len(changes) != 1 || changes[0] != "SP-1" {
		t.Fatalf("the sample holds %v, want exactly SP-1", changes)
	}
}

// TestTheSampleProofIsRunnable asserts that the plan names a command, so the
// workflow run has something to prove.
func TestTheSampleProofIsRunnable(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(sampleRoot(t), ".shipproof", "changes", "SP-1", "verification.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan struct {
		Requirements []struct {
			ID    string `json:"id"`
			Proof []struct {
				Command string `json:"command"`
			} `json:"proof"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Requirements) != 1 {
		t.Fatalf("the plan holds %d requirements, want 1", len(plan.Requirements))
	}
	if len(plan.Requirements[0].Proof) == 0 || plan.Requirements[0].Proof[0].Command == "" {
		t.Fatal("the sample requirement names no proof command")
	}
}

// TestTheSamplePackIsComplete asserts that the committed pack answers to the
// current schema. A sample that no longer validates teaches the wrong shape.
func TestTheSamplePackIsComplete(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(sampleRoot(t), ".shipproof", "changes", "SP-1", "evidence-pack.json"))
	if err != nil {
		t.Skip("the sample holds no pack yet")
	}
	var pack schema.EvidencePack
	if err := json.Unmarshal(data, &pack); err != nil {
		t.Fatal(err)
	}
	if err := pack.Validate(); err != nil {
		t.Fatalf("the sample pack does not validate: %v", err)
	}
}

// TestTheActionAcceptsAWorkingDirectory holds requirement R2 of SP-038.
func TestTheActionAcceptsAWorkingDirectory(t *testing.T) {
	body := readAction(t)
	if !strings.Contains(body, "working-directory:") {
		t.Fatal("the action names no working-directory input")
	}
	// R3. A directory with no open change must stop without failing.
	if !strings.Contains(body, "found=false") {
		t.Fatal("the action never reports a missing change")
	}
	if !strings.Contains(body, "No ShipProof change exists here") {
		t.Fatal("the action does not say why it stopped")
	}
}
