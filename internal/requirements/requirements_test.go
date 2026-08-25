package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	set := Set{
		SchemaVersion: SchemaVersion,
		ChangeID:      "SP-028",
		Adopter:       "native",
		SourcePath:    "docs/changes/SP-028.md",
		Requirements: []Requirement{{
			ID:           "SP-028-R1",
			Statement:    "Return the phase for a change directory.",
			SourceAnchor: "### SP-028-R1",
			Provenance:   Observed,
		}},
	}

	path, err := Save(root, set)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if path != Path(root, "SP-028") {
		t.Fatalf("Save() path = %q, want %q", path, Path(root, "SP-028"))
	}

	loaded, err := Load(root, "SP-028")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Requirements) != 1 {
		t.Fatalf("Requirements = %d, want 1", len(loaded.Requirements))
	}
	if loaded.Requirements[0].ID != "SP-028-R1" {
		t.Fatalf("ID = %q, want SP-028-R1", loaded.Requirements[0].ID)
	}
	if loaded.Requirements[0].Provenance != Observed {
		t.Fatalf("Provenance = %q, want observed", loaded.Requirements[0].Provenance)
	}
}

func TestSaveWritesTrailingNewline(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	set := Set{
		SchemaVersion: SchemaVersion,
		ChangeID:      "SP-028",
		Adopter:       "native",
		Requirements:  []Requirement{{ID: "SP-028-R1", Statement: "A", Provenance: Observed}},
	}
	if _, err := Save(root, set); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(Path(root, "SP-028"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "}\n") {
		t.Fatalf("file does not end with a newline:\n%q", string(data))
	}
}

func TestValidateRejectsAnEmptyChangeID(t *testing.T) {
	t.Parallel()

	set := Set{SchemaVersion: SchemaVersion, Adopter: "native",
		Requirements: []Requirement{{ID: "R1", Statement: "A", Provenance: Observed}}}
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
}

func TestValidateRejectsADuplicateIdentifier(t *testing.T) {
	t.Parallel()

	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-028", Adopter: "native",
		Requirements: []Requirement{
			{ID: "SP-028-R1", Statement: "A", Provenance: Observed},
			{ID: "SP-028-R1", Statement: "B", Provenance: Observed},
		},
	}
	err := set.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error = %v, want it to name the duplicate", err)
	}
}

func TestValidateRejectsAnUnknownProvenance(t *testing.T) {
	t.Parallel()

	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-028", Adopter: "native",
		Requirements: []Requirement{{ID: "SP-028-R1", Statement: "A", Provenance: "inferred"}},
	}
	err := set.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
	if !strings.Contains(err.Error(), "inferred") {
		t.Fatalf("error = %v, want it to name the value", err)
	}
}

func TestValidateRejectsAnEmptyRequirementSet(t *testing.T) {
	t.Parallel()

	set := Set{SchemaVersion: SchemaVersion, ChangeID: "SP-028", Adopter: "native"}
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
}

func TestValidateRejectsAnUnknownAdopter(t *testing.T) {
	t.Parallel()

	set := Set{
		SchemaVersion: SchemaVersion, ChangeID: "SP-028", Adopter: "guess",
		Requirements: []Requirement{{ID: "SP-028-R1", Statement: "A", Provenance: Observed}},
	}
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
}

func TestLoadReportsAMissingSidecar(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Load(root, "SP-028"); err == nil {
		t.Fatal("Load() = nil, want an error")
	}
}

func TestPathIsUnderTheChangeDirectory(t *testing.T) {
	t.Parallel()

	got := Path("/repo", "SP-028")
	want := filepath.Join("/repo", ".shipproof", "changes", "SP-028", "requirements.json")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}
