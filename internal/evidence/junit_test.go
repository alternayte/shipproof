package evidence

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
)

func TestParseJUnitSingleSuite(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "junit-single.xml")
	checks, err := ParseJUnit(path)
	if err != nil {
		t.Fatalf("ParseJUnit: %v", err)
	}
	if len(checks) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(checks))
	}
}

func TestParseJUnitMultipleSuites(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "junit-multiple.xml")
	checks, err := ParseJUnit(path)
	if err != nil {
		t.Fatalf("ParseJUnit: %v", err)
	}
	if len(checks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(checks))
	}
}

func TestParseJUnitCheckMapping(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "junit-single.xml")
	checks, err := ParseJUnit(path)
	if err != nil {
		t.Fatalf("ParseJUnit: %v", err)
	}

	statuses := map[string]string{}
	for _, c := range checks {
		statuses[c.ID] = c.Status
	}

	if statuses["com.example.TestSuite.testPasses"] != "pass" {
		t.Errorf("testPasses: expected pass, got %s", statuses["com.example.TestSuite.testPasses"])
	}
	if statuses["com.example.TestSuite.testFails"] != "fail" {
		t.Errorf("testFails: expected fail, got %s", statuses["com.example.TestSuite.testFails"])
	}
	if statuses["com.example.TestSuite.testErrors"] != "fail" {
		t.Errorf("testErrors: expected fail, got %s", statuses["com.example.TestSuite.testErrors"])
	}

	for _, c := range checks {
		if c.Source != "junit" {
			t.Errorf("check %s: expected source junit, got %s", c.ID, c.Source)
		}
		if c.Provenance != schema.ProvenanceObserved {
			t.Errorf("check %s: expected provenance observed, got %s", c.ID, c.Provenance)
		}
		if c.ID == "" {
			t.Errorf("check has empty ID")
		}
	}
}

func TestParseJUnitSkipped(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "junit-single.xml")
	checks, err := ParseJUnit(path)
	if err != nil {
		t.Fatalf("ParseJUnit: %v", err)
	}

	for _, c := range checks {
		if c.ID == "com.example.TestSuite.testSkipped" {
			if c.Status != "skip" {
				t.Errorf("testSkipped: expected skip, got %s", c.Status)
			}
			return
		}
	}
	t.Fatal("skipped test case not found")
}

func TestParseJUnitNeverPassOnFailure(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "junit-failures-only.xml")
	checks, err := ParseJUnit(path)
	if err != nil {
		t.Fatalf("ParseJUnit: %v", err)
	}
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	for _, c := range checks {
		if c.Status == "pass" {
			t.Errorf("check %s: expected non-pass status for failure-only suite, got pass", c.ID)
		}
	}
}

func fixturePath(t *testing.T, filename string) string {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "testdata", "evidence", filename))
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "temp.xml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
