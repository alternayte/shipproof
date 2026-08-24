package pack

import (
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
	"github.com/alternayte/shipproof/internal/schema"
)

// P12: adversarial reviewer findings enter the evidence pack as
// agent-inferred.
func TestAssembleAddsAgentReviewFindingsAsInferred(t *testing.T) {
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

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	if pack.AgentReview == nil || len(pack.AgentReview.Findings) != 1 {
		t.Fatalf("agent review = %+v", pack.AgentReview)
	}
	finding := pack.AgentReview.Findings[0]
	if finding.Provenance != schema.ProvenanceInferred {
		t.Fatalf("finding provenance = %q, want inferred", finding.Provenance)
	}
	if finding.Provenance == schema.ProvenanceObserved {
		t.Fatal("a reviewer finding must never be observed")
	}

	found := false
	for _, check := range pack.Verification.Checks {
		if check.ID == "agent:review" {
			found = true
			if check.Provenance != schema.ProvenanceInferred {
				t.Fatalf("check provenance = %q", check.Provenance)
			}
		}
	}
	if !found {
		t.Fatalf("agent:review check missing: %+v", pack.Verification.Checks)
	}
}

func TestPackRejectsObservedAgentFinding(t *testing.T) {
	base := schema.EvidencePack{
		SchemaVersion: schema.CurrentVersion,
		ChangeID:      "SP-005",
		Intent:        schema.IntentEvidence{SnapshotHash: "abc123"},
		Provenance:    schema.PackProvenance{GeneratedAt: "2026-08-24T10:00:00Z", ShipProofVersion: "0.1"},
		AgentReview: &schema.AgentReviewEvidence{
			Findings: []schema.AgentFinding{{Source: "claude", Summary: "A defect.", Provenance: schema.ProvenanceObserved}},
		},
	}
	if err := base.Validate(); err == nil {
		t.Fatal("an observed reviewer finding must fail validation")
	}
}
