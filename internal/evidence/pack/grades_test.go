package pack

import (
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/grade"
	"github.com/alternayte/shipproof/internal/schema"
)

func gradeOf(t *testing.T, checks []schema.Check, id string) string {
	t.Helper()
	for _, check := range checks {
		if check.ID == id || strings.HasPrefix(check.ID, id) {
			return check.Grade
		}
	}
	t.Fatalf("no check named %q exists: %+v", id, checks)
	return ""
}

// TestStalenessIsObserved holds requirement R1. ShipProof compares two
// SHA-256 hashes. That is a measurement, and no agent asserted it.
func TestStalenessIsObserved(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-031", "abc123")
	setupVerificationPlan(t, root, "SP-031")

	assembled, err := Assemble(root, "SP-031", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if got := gradeOf(t, assembled.Checks, "intent:staleness"); got != string(grade.Observed) {
		t.Fatalf("intent:staleness grades %q, want %q", got, grade.Observed)
	}
}

// TestAnAgentReviewFindingStaysClaimed holds requirement R3.
func TestAnAgentReviewFindingStaysClaimed(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-031", "abc123")
	setupVerificationPlan(t, root, "SP-031")

	record := agent.ExecutionRecord{
		SchemaVersion: "0.1",
		Change:        "SP-031",
		Status:        agent.OutcomePass,
		Execution:     agent.ExecutionMeta{Runner: "claude", BaseRevision: "base1", ResultRevision: "head1", Attempt: 1},
		Findings: []agent.Finding{
			{Source: "claude", Summary: "The retry path can duplicate a delivery.", Provenance: agent.ProvenanceInferred},
		},
	}
	if _, err := agent.SaveExecution(root, record); err != nil {
		t.Fatal(err)
	}

	assembled, err := Assemble(root, "SP-031", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if got := gradeOf(t, assembled.Checks, "agent:review"); got != string(grade.Claimed) {
		t.Fatalf("agent:review grades %q, want %q", got, grade.Claimed)
	}
}

// TestACoverageCheckWithNoResultIsObserved holds requirement R2.
func TestACoverageCheckWithNoResultIsObserved(t *testing.T) {
	if got := grade.FromProvenance(checkProvenance(coverage.Unknown)); got != grade.Observed {
		t.Fatalf("a coverage row with no result grades %q, want %q", got, grade.Observed)
	}
}
