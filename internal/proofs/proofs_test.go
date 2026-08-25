package proofs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func validSet() Set {
	clean := true
	return Set{
		SchemaVersion: SchemaVersion,
		ChangeID:      "SP-021",
		HeadRev:       "1cceb33",
		TreeClean:     &clean,
		Timestamp:     "2026-08-25T10:00:00Z",
		Results: []Result{
			{RequirementID: "SP-021-R1", ProofIndex: 0, Command: "go test ./internal/verification/", ExitCode: 0, DurationMs: 812, Status: Pass},
			{RequirementID: "SP-021-R6", ProofIndex: 1, Status: Human},
		},
	}
}

func TestSetValidateAcceptsAWellFormedSet(t *testing.T) {
	t.Parallel()

	if err := validSet().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSetValidateRejectsAWrongSchemaVersion(t *testing.T) {
	t.Parallel()

	set := validSet()
	set.SchemaVersion = "0.2"
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error")
	}
}

func TestSetValidateRejectsAnUnknownStatus(t *testing.T) {
	t.Parallel()

	set := validSet()
	set.Results[0].Status = "inferred"
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error for an unknown status")
	}
}

func TestSetValidateRejectsAHumanResultWithACommand(t *testing.T) {
	t.Parallel()

	set := validSet()
	set.Results[1].Command = "go test ./..."
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error; a human proof runs no command")
	}
}

func TestSetValidateRejectsAPassWithANonZeroExitCode(t *testing.T) {
	t.Parallel()

	set := validSet()
	set.Results[0].ExitCode = 1
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error; pass and a non-zero exit code disagree")
	}
}

func TestSetValidateRejectsADuplicateResult(t *testing.T) {
	t.Parallel()

	set := validSet()
	set.Results[1] = set.Results[0]
	if err := set.Validate(); err == nil {
		t.Fatal("Validate() = nil, want an error for a duplicate requirement and proof index")
	}
}

func TestSetSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path, err := Save(root, validSet())
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if want := Path(root, "SP-021"); path != want {
		t.Fatalf("Save() path = %q, want %q", path, want)
	}
	if !Exists(root, "SP-021") {
		t.Fatal("Exists() = false after Save()")
	}

	loaded, err := Load(root, "SP-021")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Results) != 2 {
		t.Fatalf("Results = %d, want 2", len(loaded.Results))
	}
	if loaded.Results[0].Status != Pass {
		t.Fatalf("Status = %q, want %q", loaded.Results[0].Status, Pass)
	}
}

func TestSetSaveWritesUnderTheRunDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Save(root, validSet()); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".shipproof", "runs", "SP-021", "proofs.json")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("stat %s: %v", want, err)
	}
}

func TestSetSaveWritesTheDocumentedShape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path, err := Save(root, validSet())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "change_id", "head_rev", "tree_clean", "timestamp", "results"} {
		if _, present := document[key]; !present {
			t.Fatalf("proofs.json has no %q key", key)
		}
	}
}

func TestLoadRejectsAnAbsentArtifact(t *testing.T) {
	t.Parallel()

	if _, err := Load(t.TempDir(), "SP-021"); err == nil {
		t.Fatal("Load() = nil error, want an error for an absent artifact")
	}
}
