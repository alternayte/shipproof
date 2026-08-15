package evidence

import (
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
)

func TestParseSARIF(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "sarif-errors.json")
	checks, err := ParseSARIF(path)
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
}

func TestParseSARIFMultipleResults(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "sarif-errors.json")
	checks, err := ParseSARIF(path)
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}

	ids := map[string]bool{}
	for _, c := range checks {
		ids[c.ID] = true
	}
	if !ids["G104"] {
		t.Error("expected G104 result")
	}
	if !ids["G401"] {
		t.Error("expected G401 result")
	}
}

func TestParseSARIFLevelMapping(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "sarif-all-levels.json")
	checks, err := ParseSARIF(path)
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}

	expectedStatus := map[string]string{
		"errorsas": "fail",
		"printf":   "unknown",
		"shadow":   "unknown",
		"nilfunc":  "skip",
	}
	for _, c := range checks {
		want, ok := expectedStatus[c.ID]
		if !ok {
			t.Errorf("unexpected check ID: %s", c.ID)
			continue
		}
		if c.Status != want {
			t.Errorf("check %s: expected status %s, got %s", c.ID, want, c.Status)
		}
		if c.Source != "sarif" {
			t.Errorf("check %s: expected source sarif, got %s", c.ID, c.Source)
		}
		if c.Provenance != schema.ProvenanceObserved {
			t.Errorf("check %s: expected provenance observed, got %s", c.ID, c.Provenance)
		}
	}
}

func TestParseSARIFMissingRuleID(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "sarif-missing-ruleid.json")
	checks, err := ParseSARIF(path)
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}

	if checks[0].ID != "result-0" {
		t.Errorf("first check: expected result-0 for missing ruleId, got %s", checks[0].ID)
	}
	if checks[1].ID != "C001" {
		t.Errorf("second check: expected C001, got %s", checks[1].ID)
	}
}

func TestParseSARIFNeverPassOnError(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "sarif-errors.json")
	checks, err := ParseSARIF(path)
	if err != nil {
		t.Fatalf("ParseSARIF: %v", err)
	}
	for _, c := range checks {
		if c.Status == "pass" {
			t.Errorf("check %s: expected non-pass for error-level SARIF result, got pass", c.ID)
		}
	}
}
