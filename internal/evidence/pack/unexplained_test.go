package pack

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/schema"
	"github.com/alternayte/shipproof/internal/verification"
)

// TestCoverageChecksCarryTheMatrixDetail confirms an awaiting-human row and
// an unproven row, which share the same status and provenance, still read
// apart through the detail a reviewer sees.
func TestCoverageChecksCarryTheMatrixDetail(t *testing.T) {
	matrix := coverage.Matrix{
		ChangeID: "SP-900",
		Rows: []coverage.Row{
			{RequirementID: "SP-900-R1", State: coverage.AwaitingHuman, Provenance: coverage.Unknown,
				Detail: "1 of 1 human proofs carry no recorded acceptance"},
			{RequirementID: "SP-900-R2", State: coverage.Unproven, Provenance: coverage.Unknown,
				Detail: "the plan entry names no usable proof"},
		},
	}

	checks := coverageChecks(matrix)
	if len(checks) != 2 {
		t.Fatalf("checks = %d, want 2", len(checks))
	}
	for _, check := range checks {
		if check.Status != "unknown" {
			t.Errorf("check %q status = %q, want unknown", check.ID, check.Status)
		}
	}
	if checks[0].Detail == checks[1].Detail {
		t.Fatalf("awaiting-human and unproven checks carry the same detail: %q", checks[0].Detail)
	}
	if checks[0].Detail != matrix.Rows[0].Detail {
		t.Errorf("checks[0].Detail = %q, want %q", checks[0].Detail, matrix.Rows[0].Detail)
	}
	if checks[1].Detail != matrix.Rows[1].Detail {
		t.Errorf("checks[1].Detail = %q, want %q", checks[1].Detail, matrix.Rows[1].Detail)
	}
}

// TestBuildUnexplainedMarshalsEmptyFindingsAsArrays confirms a change with no
// unexplained finding marshals both finding fields as JSON arrays, never
// null, so the section validates against the schema's array-typed properties.
func TestBuildUnexplainedMarshalsEmptyFindingsAsArrays(t *testing.T) {
	root := t.TempDir()
	initGitRepo(t, root)
	rev := writeAndCommit(t, root, "unchanged.txt", "content")

	section := buildUnexplained(root, "SP-901", verification.Plan{}, rev, rev)
	if section == nil {
		t.Fatal("buildUnexplained returned nil with a valid revision range")
	}
	if len(section.LineFindings) != 0 || len(section.FileFindings) != 0 {
		t.Fatalf("expected no findings, got %d line and %d file findings", len(section.LineFindings), len(section.FileFindings))
	}

	pack := schema.EvidencePack{
		SchemaVersion:     "0.1",
		ChangeID:          "SP-901",
		Intent:            schema.IntentEvidence{SnapshotHash: "abc123"},
		Verification:      schema.VerificationEvidence{Checks: []schema.Check{}},
		Provenance:        schema.PackProvenance{GeneratedAt: "2026-08-14T20:00:00Z", ShipProofVersion: "0.1"},
		UnexplainedChange: section,
	}
	if err := pack.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	data, err := json.Marshal(pack)
	if err != nil {
		t.Fatalf("marshal pack: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, `"line_findings":[]`) {
		t.Errorf("marshaled pack has no empty line_findings array: %s", body)
	}
	if !strings.Contains(body, `"file_findings":[]`) {
		t.Errorf("marshaled pack has no empty file_findings array: %s", body)
	}
	if strings.Contains(body, `"line_findings":null`) || strings.Contains(body, `"file_findings":null`) {
		t.Fatalf("marshaled pack holds null instead of an empty array: %s", body)
	}
}
