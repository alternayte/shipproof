package pack

import (
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
	"github.com/alternayte/shipproof/internal/grade"
	"github.com/alternayte/shipproof/internal/schema"
)

// TestAnAgentReviewFindingIsAClaimedCheck holds the honesty rule. Section 7
// names no agent-review section. A reviewer finding is an agent claim, so it
// enters the check list with a claimed grade. It is never observed.
func TestAnAgentReviewFindingIsAClaimedCheck(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	record := agent.ExecutionRecord{
		SchemaVersion: "0.1",
		Change:        "SP-005",
		Status:        agent.OutcomePass,
		Execution:     agent.ExecutionMeta{Runner: "claude", BaseRevision: "base1", ResultRevision: "head1", Attempt: 1},
		Findings: []agent.Finding{
			{Source: "claude", Summary: "The retry path can duplicate a delivery.", Provenance: agent.ProvenanceInferred},
		},
	}
	if _, err := agent.SaveExecution(root, record); err != nil {
		t.Fatalf("save execution record: %v", err)
	}

	assembled, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	found := false
	for _, check := range assembled.Checks {
		if !strings.HasPrefix(check.ID, "agent:review") {
			continue
		}
		found = true
		if got := grade.FromProvenance(check.Provenance); got != grade.Claimed {
			t.Fatalf("check %q carries the grade %q, want %q", check.ID, got, grade.Claimed)
		}
		if check.Provenance == schema.ProvenanceObserved {
			t.Fatal("a reviewer finding must never be observed")
		}
		if check.Detail != "The retry path can duplicate a delivery." {
			t.Fatalf("check detail = %q", check.Detail)
		}
	}
	if !found {
		t.Fatalf("no agent review check exists: %+v", assembled.Checks)
	}
}
