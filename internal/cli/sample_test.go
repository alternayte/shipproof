package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/requirements"
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

// TestTheSampleChangeRecordIsLoadable asserts the committed inputs of the
// sample. It does not read the generated pack: that file is not committed, so
// a test that read it would pass on a build that never wrote one and fail on a
// machine that holds a stale copy. That is the defect that turned the build
// red once already.
func TestTheSampleChangeRecordIsLoadable(t *testing.T) {
	root := sampleRoot(t)
	record, err := change.Load(root, "SP-1")
	if err != nil {
		t.Fatalf("load the sample change record: %v", err)
	}
	if record.SourcePath == "" || record.SHA256 == "" {
		t.Fatalf("the sample change record is incomplete: %+v", record)
	}
	set, err := requirements.Load(root, "SP-1")
	if err != nil {
		t.Fatalf("load the sample requirement set: %v", err)
	}
	if len(set.Requirements) == 0 {
		t.Fatal("the sample holds no requirement")
	}
	for _, requirement := range set.Requirements {
		if requirement.SourceAnchor == "" {
			t.Errorf("requirement %s carries no source anchor, so the report cannot show where it came from",
				requirement.ID)
		}
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
