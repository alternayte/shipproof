package pack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// section7Fields is the complete top-level field set of Section 7, plus the
// two fields that the rules of Section 7 need: `empty_sections` for rule 2 and
// `provenance` for rule 3.
var section7Fields = []string{
	"agent",
	"attestation",
	"change_id",
	"checks",
	"empty_sections",
	"implementation",
	"intent",
	"provenance",
	"requirements",
	"schema_version",
	"unexplained_change",
	"verdict",
}

func assembleForShape(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-027", "abc123")
	setupVerificationPlan(t, root, "SP-027")

	assembled, err := Assemble(root, "SP-027", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	data, err := json.Marshal(assembled)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return fields
}

// TestThePackHoldsTheSection7Fields asserts the exact top-level key set.
func TestThePackHoldsTheSection7Fields(t *testing.T) {
	fields := assembleForShape(t)

	got := make([]string, 0, len(fields))
	for name := range fields {
		got = append(got, name)
	}
	sort.Strings(got)

	if len(got) != len(section7Fields) {
		t.Fatalf("the pack holds the fields %v, want %v", got, section7Fields)
	}
	for index, name := range got {
		if name != section7Fields[index] {
			t.Fatalf("the pack holds the fields %v, want %v", got, section7Fields)
		}
	}
}

// TestTheCutSectionsAreGone holds requirement R11.
func TestTheCutSectionsAreGone(t *testing.T) {
	fields := assembleForShape(t)
	for _, name := range []string{"readiness", "review", "agent_review", "agent_run", "verification"} {
		if _, present := fields[name]; present {
			t.Fatalf("the pack still holds the cut section %q", name)
		}
	}
}

// TestEveryEmptySectionStatesItsReason holds rule 2 of Section 7.
func TestEveryEmptySectionStatesItsReason(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-027", "abc123")
	setupVerificationPlan(t, root, "SP-027")

	assembled, err := Assemble(root, "SP-027", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	// This run holds no telemetry record, no base revision, and no signature.
	for _, name := range []string{"agent", "unexplained_change", "attestation"} {
		reason, stated := assembled.EmptySections[name]
		if !stated {
			t.Fatalf("the section %q is empty and the pack states no reason", name)
		}
		if reason == "" {
			t.Fatalf("the section %q holds an empty reason", name)
		}
	}
}

// TestThePackCarriesTheVerdict holds requirement R2.
func TestThePackCarriesTheVerdict(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-027", "abc123")
	setupVerificationPlan(t, root, "SP-027")

	assembled, err := Assemble(root, "SP-027", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if assembled.Verdict.Verdict == "" {
		t.Fatal("the pack states no verdict")
	}
	if assembled.Verdict.Reason == "" || assembled.Verdict.Next == "" {
		t.Fatalf("the verdict is incomplete: %+v", assembled.Verdict)
	}
}

// TestARequirementRowCarriesItsProofResult holds requirement R4.
func TestARequirementRowCarriesItsProofResult(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-027", "abc123")
	setupVerificationPlan(t, root, "SP-027")

	assembled, err := Assemble(root, "SP-027", Options{})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if len(assembled.Requirements) != 2 {
		t.Fatalf("the pack holds %d requirement rows, want 2", len(assembled.Requirements))
	}
	for _, row := range assembled.Requirements {
		if row.ID == "" {
			t.Fatal("a requirement row carries no identifier")
		}
		if row.State == "" {
			t.Fatalf("requirement %q carries no state", row.ID)
		}
		if row.Grade == "" {
			t.Fatalf("requirement %q carries no grade", row.ID)
		}
	}
}

// TestAFailedStageWritesNoFile is the proof for row C3 of the definition of
// done.
func TestAFailedStageWritesNoFile(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-027", "abc123")
	// The verification plan is missing. That stage fails.

	assembled, err := Assemble(root, "SP-027", Options{})
	if err == nil {
		t.Fatal("assemble returned no error for a missing verification plan")
	}
	if writeErr := WritePack(root, assembled); writeErr == nil {
		t.Fatal("WritePack accepted an incomplete pack")
	}
	path := filepath.Join(root, ".shipproof", "changes", "SP-027", "evidence-pack.json")
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("a partial pack exists at %s", path)
	}
}
