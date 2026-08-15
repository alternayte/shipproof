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
	path := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "schemas", "v0.1", "evidence.schema.json"))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(contents, &document); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
}

func TestReadinessEvidence(t *testing.T) {
	base := EvidencePack{
		SchemaVersion: "0.1",
		ChangeID:      "SP-012",
		Intent:        IntentEvidence{SnapshotHash: "abc123"},
		Verification:  VerificationEvidence{Checks: []Check{}},
		Provenance:    PackProvenance{GeneratedAt: "2026-08-14T20:00:00Z", ShipProofVersion: "0.1"},
	}

	withReadiness := base
	withReadiness.Readiness = &ReadinessEvidence{ShapingRef: "complete-metrics", BlockerCount: 2}
	if err := withReadiness.Validate(); err != nil {
		t.Fatalf("pack with readiness must validate: %v", err)
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("pack without readiness must validate: %v", err)
	}

	missingRef := base
	missingRef.Readiness = &ReadinessEvidence{BlockerCount: 2}
	if err := missingRef.Validate(); err == nil {
		t.Fatal("readiness without shaping_ref must fail validation")
	}

	badRef := base
	badRef.Readiness = &ReadinessEvidence{ShapingRef: "Bad Ref!", BlockerCount: 2}
	if err := badRef.Validate(); err == nil {
		t.Fatal("readiness with invalid shaping_ref format must fail validation")
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
