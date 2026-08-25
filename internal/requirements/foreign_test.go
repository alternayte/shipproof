package requirements

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsNativeAcceptsANativeDocument(t *testing.T) {
	t.Parallel()

	native, err := IsNative("SP-011", filepath.Join("testdata", "SP-011-native.md"))
	if err != nil {
		t.Fatalf("IsNative() error = %v", err)
	}
	if !native {
		t.Fatal("IsNative() = false, want true")
	}
}

func TestIsNativeRejectsAForeignDocument(t *testing.T) {
	t.Parallel()

	native, err := IsNative("SP-050", filepath.Join("testdata", "foreign.md"))
	if err != nil {
		t.Fatalf("IsNative() error = %v", err)
	}
	if native {
		t.Fatal("IsNative() = true, want false")
	}
}

func TestIsNativeReportsAMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := IsNative("SP-050", filepath.Join("testdata", "absent.md")); err == nil {
		t.Fatal("IsNative() = nil error, want an error")
	}
}

func TestProposeForeignExtractsCandidates(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join("testdata", "foreign.md"))
	if err != nil {
		t.Fatal(err)
	}

	set, err := ProposeForeign("SP-050", "testdata/foreign.md", body)
	if err != nil {
		t.Fatalf("ProposeForeign() error = %v", err)
	}
	if set.Adopter != AdopterForeign {
		t.Fatalf("Adopter = %q, want %q", set.Adopter, AdopterForeign)
	}

	want := []string{
		"Retry a failed charge",
		"Cap the retry count",
		"MUST stop after five attempts.",
		"Record every attempt",
	}
	if len(set.Requirements) != len(want) {
		t.Fatalf("Requirements = %d, want %d: %+v", len(set.Requirements), len(want), set.Requirements)
	}
	for index, statement := range want {
		got := set.Requirements[index]
		if got.Statement != statement {
			t.Fatalf("requirement %d statement = %q, want %q", index+1, got.Statement, statement)
		}
		wantID := fmt.Sprintf("SP-050-R%d", index+1)
		if got.ID != wantID {
			t.Fatalf("requirement %d id = %q, want %q", index+1, got.ID, wantID)
		}
		if got.Provenance != Human {
			t.Fatalf("requirement %d provenance = %q, want %q", index+1, got.Provenance, Human)
		}
		if got.ConfirmedAt != "" {
			t.Fatalf("requirement %d is confirmed before a person confirmed it", index+1)
		}
	}
}

func TestProposeForeignSkipsTheTitleHeading(t *testing.T) {
	t.Parallel()

	body := []byte("# Title\n\n## Only requirement\n")
	set, err := ProposeForeign("SP-050", "x.md", body)
	if err != nil {
		t.Fatalf("ProposeForeign() error = %v", err)
	}
	if len(set.Requirements) != 1 {
		t.Fatalf("Requirements = %d, want 1", len(set.Requirements))
	}
	if set.Requirements[0].Statement != "Only requirement" {
		t.Fatalf("Statement = %q", set.Requirements[0].Statement)
	}
}

func TestProposeForeignRejectsADocumentWithNoCandidate(t *testing.T) {
	t.Parallel()

	body := []byte("Plain prose with no heading and no obligation.\n")
	if _, err := ProposeForeign("SP-050", "x.md", body); err == nil {
		t.Fatal("ProposeForeign() = nil, want an error")
	}
}

func TestProposeForeignIgnoresAFencedBlock(t *testing.T) {
	t.Parallel()

	body := []byte("```\n## Not a requirement\n```\n\n## A requirement\n")
	set, err := ProposeForeign("SP-050", "x.md", body)
	if err != nil {
		t.Fatalf("ProposeForeign() error = %v", err)
	}
	if len(set.Requirements) != 1 {
		t.Fatalf("Requirements = %d, want 1", len(set.Requirements))
	}
	if set.Requirements[0].Statement != "A requirement" {
		t.Fatalf("Statement = %q", set.Requirements[0].Statement)
	}
}

func TestRequiresConfirmationHoldsForAnUnstampedForeignSet(t *testing.T) {
	t.Parallel()

	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-050", Adopter: AdopterForeign,
		Requirements: []Requirement{{ID: "SP-050-R1", Statement: "A", Provenance: Human}},
	}
	if !set.RequiresConfirmation() {
		t.Fatal("RequiresConfirmation() = false, want true")
	}
}

func TestRequiresConfirmationIsFalseForANativeSet(t *testing.T) {
	t.Parallel()

	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-011", Adopter: AdopterNative,
		Requirements: []Requirement{{ID: "SP-011-R1", Statement: "A", Provenance: Observed}},
	}
	if set.RequiresConfirmation() {
		t.Fatal("RequiresConfirmation() = true, want false")
	}
}

func TestConfirmStampsEveryHumanRequirement(t *testing.T) {
	t.Parallel()

	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-050", Adopter: AdopterForeign,
		Requirements: []Requirement{
			{ID: "SP-050-R1", Statement: "A", Provenance: Human},
			{ID: "SP-050-R2", Statement: "B", Provenance: Human},
		},
	}
	moment := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	confirmed := set.Confirm(moment)

	if confirmed.RequiresConfirmation() {
		t.Fatal("RequiresConfirmation() = true after Confirm()")
	}
	for _, requirement := range confirmed.Requirements {
		if requirement.ConfirmedAt != "2026-08-25T10:00:00Z" {
			t.Fatalf("ConfirmedAt = %q, want 2026-08-25T10:00:00Z", requirement.ConfirmedAt)
		}
	}
	if set.Requirements[0].ConfirmedAt != "" {
		t.Fatal("Confirm() mutated the receiver")
	}
}

func TestErrUnconfirmedIsExported(t *testing.T) {
	t.Parallel()

	if !errors.Is(ErrUnconfirmed, ErrUnconfirmed) {
		t.Fatal("ErrUnconfirmed is not usable with errors.Is")
	}
}

func TestSaveRefusesAnUnconfirmedForeignSet(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-050", Adopter: AdopterForeign,
		Requirements: []Requirement{{ID: "SP-050-R1", Statement: "A", Provenance: Human}},
	}
	_, err := Save(root, set)
	if !errors.Is(err, ErrUnconfirmed) {
		t.Fatalf("Save() error = %v, want ErrUnconfirmed", err)
	}
	if _, statErr := os.Stat(Path(root, "SP-050")); statErr == nil {
		t.Fatal("Save() wrote the sidecar despite the refusal")
	}
}

func TestSaveAcceptsAConfirmedForeignSet(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-050", Adopter: AdopterForeign,
		Requirements: []Requirement{{ID: "SP-050-R1", Statement: "A", Provenance: Human}},
	}.Confirm(time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC))

	if _, err := Save(root, set); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load(root, "SP-050")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Requirements[0].ConfirmedAt != "2026-08-25T10:00:00Z" {
		t.Fatalf("ConfirmedAt = %q", loaded.Requirements[0].ConfirmedAt)
	}
}
