package skills

import "testing"

func TestBuiltInEvalManifestIsValid(t *testing.T) {
	manifest, err := LoadBuiltInEvals()
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Cases) < 5 {
		t.Fatalf("expected at least 5 eval cases, got %d", len(manifest.Cases))
	}
}

func TestValidateEvalManifestRejectsDuplicateIDs(t *testing.T) {
	manifest := EvalManifest{SchemaVersion: "0.1", Cases: []EvalCase{
		{ID: "same", Skill: "shape-prd", Goal: "one", Expected: []string{"x"}},
		{ID: "same", Skill: "shape-prd", Goal: "two", Expected: []string{"x"}},
	}}
	if err := ValidateEvalManifest(manifest); err == nil {
		t.Fatal("expected duplicate id error")
	}
}
