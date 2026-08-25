package requirements

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseNativeGoldenSP011(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join("testdata", "SP-011-native.md"))
	if err != nil {
		t.Fatal(err)
	}

	set, err := ParseNative("SP-011", "docs/changes/SP-011.md", body)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}
	if set.Adopter != AdopterNative {
		t.Fatalf("Adopter = %q, want %q", set.Adopter, AdopterNative)
	}
	if len(set.Requirements) != 8 {
		t.Fatalf("Requirements = %d, want 8", len(set.Requirements))
	}

	want := []Requirement{
		{ID: "SP-011-R1", Statement: "Derive per-change cycle time", SourceAnchor: "### SP-011-R1 — Derive per-change cycle time", Provenance: Observed},
		{ID: "SP-011-R2", Statement: "Derive project average cycle time", SourceAnchor: "### SP-011-R2 — Derive project average cycle time", Provenance: Observed},
		{ID: "SP-011-R3", Statement: "Derive per-change rework", SourceAnchor: "### SP-011-R3 — Derive per-change rework", Provenance: Observed},
		{ID: "SP-011-R4", Statement: "Derive project average rework", SourceAnchor: "### SP-011-R4 — Derive project average rework", Provenance: Observed},
		{ID: "SP-011-R5", Statement: "Render metric cards", SourceAnchor: "### SP-011-R5 — Render metric cards", Provenance: Observed},
		{ID: "SP-011-R6", Statement: "Extend summary table", SourceAnchor: "### SP-011-R6 — Extend summary table", Provenance: Observed},
		{ID: "SP-011-R7", Statement: "Remove unavailable placeholders", SourceAnchor: "### SP-011-R7 — Remove unavailable placeholders", Provenance: Observed},
		{ID: "SP-011-R8", Statement: "Tests", SourceAnchor: "### SP-011-R8 — Tests", Provenance: Observed},
	}
	for index, expected := range want {
		got := set.Requirements[index]
		if got != expected {
			t.Fatalf("requirement %d = %+v, want %+v", index+1, got, expected)
		}
	}
	if err := set.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestParseNativeAcceptsAHyphenSeparator(t *testing.T) {
	t.Parallel()

	body := []byte("## Requirements\n\n### SP-028-R1 - Return the phase\n\nBody.\n")
	set, err := ParseNative("SP-028", "docs/changes/SP-028.md", body)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}
	if len(set.Requirements) != 1 {
		t.Fatalf("Requirements = %d, want 1", len(set.Requirements))
	}
	if set.Requirements[0].Statement != "Return the phase" {
		t.Fatalf("Statement = %q", set.Requirements[0].Statement)
	}
}

func TestParseNativeIgnoresAnotherChangeIdentifier(t *testing.T) {
	t.Parallel()

	body := []byte("### SP-028-R1 — Mine\n\n### SP-029-R1 — Not mine\n")
	set, err := ParseNative("SP-028", "docs/changes/SP-028.md", body)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}
	if len(set.Requirements) != 1 {
		t.Fatalf("Requirements = %d, want 1", len(set.Requirements))
	}
	if set.Requirements[0].ID != "SP-028-R1" {
		t.Fatalf("ID = %q, want SP-028-R1", set.Requirements[0].ID)
	}
}

func TestParseNativeIgnoresAHeadingInAFencedBlock(t *testing.T) {
	t.Parallel()

	body := []byte("```markdown\n### SP-028-R9 — Example only\n```\n\n### SP-028-R1 — Real\n")
	set, err := ParseNative("SP-028", "docs/changes/SP-028.md", body)
	if err != nil {
		t.Fatalf("ParseNative() error = %v", err)
	}
	if len(set.Requirements) != 1 {
		t.Fatalf("Requirements = %d, want 1", len(set.Requirements))
	}
	if set.Requirements[0].ID != "SP-028-R1" {
		t.Fatalf("ID = %q, want SP-028-R1", set.Requirements[0].ID)
	}
}

func TestParseNativeRejectsADocumentWithNoRequirement(t *testing.T) {
	t.Parallel()

	body := []byte("# A plain document\n\nNo requirement heading.\n")
	_, err := ParseNative("SP-028", "docs/changes/SP-028.md", body)
	if !errors.Is(err, ErrNoNativeRequirement) {
		t.Fatalf("error = %v, want ErrNoNativeRequirement", err)
	}
}

func TestParseNativeRejectsADuplicateHeading(t *testing.T) {
	t.Parallel()

	body := []byte("### SP-028-R1 — One\n\n### SP-028-R1 — Two\n")
	if _, err := ParseNative("SP-028", "docs/changes/SP-028.md", body); err == nil {
		t.Fatal("ParseNative() = nil, want an error")
	}
}

func TestParseNativeRejectsAHeadingWithNoTitle(t *testing.T) {
	t.Parallel()

	body := []byte("### SP-028-R1 —\n")
	if _, err := ParseNative("SP-028", "docs/changes/SP-028.md", body); err == nil {
		t.Fatal("ParseNative() = nil, want an error")
	}
}

func TestAdoptNativeReadsTheFile(t *testing.T) {
	t.Parallel()

	set, err := AdoptNative("SP-011", filepath.Join("testdata", "SP-011-native.md"))
	if err != nil {
		t.Fatalf("AdoptNative() error = %v", err)
	}
	if len(set.Requirements) != 8 {
		t.Fatalf("Requirements = %d, want 8", len(set.Requirements))
	}
	if set.SourcePath != filepath.Join("testdata", "SP-011-native.md") {
		t.Fatalf("SourcePath = %q", set.SourcePath)
	}
}

func TestAdoptNativeReportsAMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := AdoptNative("SP-011", filepath.Join("testdata", "absent.md")); err == nil {
		t.Fatal("AdoptNative() = nil, want an error")
	}
}
