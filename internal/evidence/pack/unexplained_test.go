package pack

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
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

// TestAssembleFallsBackToTheRecordedBaseRevision proves that the documented
// invocation, which passes no --base, still carries the unexplained-change
// section. The agent execution record holds the base revision the run
// started from.
func TestAssembleFallsBackToTheRecordedBaseRevision(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-902", "abc123")
	setupVerificationPlan(t, root, "SP-902")

	initGitRepo(t, root)
	base := writeAndCommit(t, root, "initial.txt", "first")
	writeAndCommit(t, root, "second.txt", "second")

	record := agent.ExecutionRecord{
		SchemaVersion: "0.1",
		Change:        "SP-902",
		Execution:     agent.ExecutionMeta{Runner: "claude", BaseRevision: base, ResultRevision: base},
	}
	if _, err := agent.SaveExecution(root, record); err != nil {
		t.Fatal(err)
	}

	var warnings strings.Builder
	assembled, err := Assemble(root, "SP-902", Options{Warn: &warnings})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if assembled.UnexplainedChange == nil {
		t.Fatalf("the section is missing, warnings = %q", warnings.String())
	}
	if warnings.Len() != 0 {
		t.Errorf("Assemble warned about a section it produced: %q", warnings.String())
	}
}

// TestAssembleReportsAnOmittedUnexplainedSection proves that a pack with no
// base revision at all says so on the warning writer. Silence is the wrong
// default for a signal whose purpose is to be read.
func TestAssembleReportsAnOmittedUnexplainedSection(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-903", "abc123")
	setupVerificationPlan(t, root, "SP-903")

	var warnings strings.Builder
	assembled, err := Assemble(root, "SP-903", Options{Warn: &warnings})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if assembled.UnexplainedChange != nil {
		t.Fatal("the section is present without any base revision")
	}
	if !strings.Contains(warnings.String(), "unexplained change") {
		t.Errorf("Assemble omitted the section in silence: %q", warnings.String())
	}
}
