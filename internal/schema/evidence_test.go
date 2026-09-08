package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEvidenceFixtureValidates(t *testing.T) {
	fixture := readFixture(t, "valid", "minimal.json")
	var pack EvidencePack
	if err := json.Unmarshal(fixture, &pack); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if err := pack.Validate(); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
}

func TestInvalidEvidenceFixtureFails(t *testing.T) {
	fixture := readFixture(t, "invalid", "missing-change-id.json")
	var pack EvidencePack
	if err := json.Unmarshal(fixture, &pack); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if err := pack.Validate(); err == nil {
		t.Fatal("expected fixture validation to fail")
	}
}

func TestJSONSchemaDocumentIsValidJSON(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	path := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "schemas", "v"+CurrentVersion, "evidence.schema.json"))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(contents, &document); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

// TestAnUnmeasuredSectionMustStateItsReason holds rule 2 of Section 7. The
// unexplained-change section is never optional. An unmeasured section states
// why it is empty, and a pack that states no reason fails.
func TestAnUnmeasuredSectionMustStateItsReason(t *testing.T) {
	pack := EvidencePack{
		SchemaVersion: CurrentVersion,
		ChangeID:      "SP-300",
		Verdict:       VerdictEvidence{Verdict: "NOT PROVEN", Reason: "no proof ran.", Next: "run `shipproof prove SP-300`."},
		Intent:        IntentEvidence{SnapshotHash: "abc123"},
		Checks:        []Check{},
		EmptySections: map[string]string{},
		Provenance:    PackProvenance{GeneratedAt: "2026-08-14T20:00:00Z", ShipProofVersion: CurrentVersion},
	}
	if err := pack.Validate(); err == nil {
		t.Fatal("Validate accepted an empty section with no stated reason")
	}

	pack.EmptySections = map[string]string{
		"requirements":       "the change holds no requirement set.",
		"checks":             "no tool result reached this pack.",
		"implementation":     "no base revision is known.",
		"unexplained_change": "ShipProof did not reach the measurement.",
		"agent":              "no telemetry record exists.",
		"attestation":        "a local pack is unsigned.",
	}
	if err := pack.Validate(); err != nil {
		t.Fatalf("Validate with every reason stated: %v", err)
	}

	pack.UnexplainedChange = UnexplainedEvidence{
		Measured:            true,
		CoverageAvailable:   true,
		UninstrumentedLines: 61,
		LineFindings:        []UnexplainedLine{{File: "a.go", Symbol: "F", StartLine: 1, EndLine: 2}},
		FileFindings:        []UnexplainedFile{{Path: "docs/a.md", IgnorePattern: "docs/**"}},
	}
	// The section now holds a measurement, so its reason must go with it.
	delete(pack.EmptySections, "unexplained_change")
	if err := pack.Validate(); err != nil {
		t.Fatalf("Validate with a measured section: %v", err)
	}
}

func readFixture(t *testing.T, category, filename string) []byte {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	path := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "testdata", "evidence", category, filename))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return contents
}

// TestAFilledSectionHoldsNoReason holds rule 2 of Section 7 in the other
// direction. empty_sections describes what is empty. A reason that survives
// after a section is filled tells a reader the opposite of the truth.
func TestAFilledSectionHoldsNoReason(t *testing.T) {
	pack := EvidencePack{
		SchemaVersion: CurrentVersion,
		ChangeID:      "SP-300",
		Verdict:       VerdictEvidence{Verdict: "NOT PROVEN", Reason: "no proof ran.", Next: "run `shipproof prove SP-300`."},
		Intent:        IntentEvidence{SnapshotHash: "abc123"},
		Checks:        []Check{},
		Attestation:   &AttestationEvidence{Format: "in-toto", Signature: "MEUCIQ"},
		EmptySections: map[string]string{
			"requirements":       "none",
			"checks":             "none",
			"implementation":     "none",
			"unexplained_change": "none",
			"agent":              "none",
			"attestation":        "a local pack is unsigned.",
		},
		Provenance: PackProvenance{GeneratedAt: "2026-09-09T10:00:00Z", ShipProofVersion: "0.4.0"},
	}
	if err := pack.Validate(); err == nil {
		t.Fatal("Validate accepted a signed pack that still calls its attestation empty")
	}

	delete(pack.EmptySections, "attestation")
	if err := pack.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
