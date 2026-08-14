package evidence

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMissingFile(t *testing.T) {
	t.Parallel()
	_, err := ParseFiles([]string{"/nonexistent/path.xml"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseInvalidXML(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, "<testsuite><notclosed>")
	_, err := ParseFiles([]string{path})
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, "not JSON")
	_, err := ParseFiles([]string{path})
	if err == nil {
		t.Fatal("expected error for file that is not valid JSON or XML")
	}
}

func TestParseUnknownFormat(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, "plain text with no markers")
	_, err := ParseFiles([]string{path})
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !errors.Is(err, ErrUnknownFormat) {
		t.Errorf("expected ErrUnknownFormat, got %v", err)
	}
}

func TestParseFilesSingleFormat(t *testing.T) {
	t.Parallel()
	paths := []string{
		fixturePath(t, "junit-single.xml"),
		fixturePath(t, "junit-multiple.xml"),
	}
	checks, err := ParseFiles(paths)
	if err != nil {
		t.Fatalf("ParseFiles: %v", err)
	}
	if len(checks) != 7 {
		t.Fatalf("expected 7 checks (4 + 3), got %d", len(checks))
	}
}

func TestParseFilesMixed(t *testing.T) {
	t.Parallel()
	paths := []string{
		fixturePath(t, "junit-single.xml"),
		fixturePath(t, "sarif-errors.json"),
	}
	checks, err := ParseFiles(paths)
	if err != nil {
		t.Fatalf("ParseFiles: %v", err)
	}
	if len(checks) != 6 {
		t.Fatalf("expected 6 checks (4 junit + 2 sarif), got %d", len(checks))
	}

	hasJUnit := false
	hasSARIF := false
	for _, c := range checks {
		switch c.Source {
		case "junit":
			hasJUnit = true
		case "sarif":
			hasSARIF = true
		}
	}
	if !hasJUnit {
		t.Error("no junit checks found")
	}
	if !hasSARIF {
		t.Error("no sarif checks found")
	}
}

func TestInvariantNoPassOnFailure(t *testing.T) {
	t.Parallel()
	paths := []string{
		fixturePath(t, "junit-failures-only.xml"),
		fixturePath(t, "sarif-errors.json"),
	}
	checks, err := ParseFiles(paths)
	if err != nil {
		t.Fatalf("ParseFiles: %v", err)
	}
	for _, c := range checks {
		failureSources := c.Status == "fail" || c.Status == "skip" || c.Status == "unknown"
		if !failureSources && c.Status == "pass" {
			t.Errorf("check %s (source %s): all fixtures contain failures/errors only, but got pass", c.ID, c.Source)
		}
		_ = failureSources
	}
}

func TestInvariantProvenance(t *testing.T) {
	t.Parallel()
	paths := []string{
		fixturePath(t, "junit-single.xml"),
		fixturePath(t, "sarif-all-levels.json"),
	}
	checks, err := ParseFiles(paths)
	if err != nil {
		t.Fatalf("ParseFiles: %v", err)
	}
	for _, c := range checks {
		if c.Provenance != "observed" {
			t.Errorf("check %s: expected provenance observed, got %s", c.ID, c.Provenance)
		}
		if c.Source != "junit" && c.Source != "sarif" {
			t.Errorf("check %s: expected source junit or sarif, got %s", c.ID, c.Source)
		}
	}
}

func TestParseEmptyFile(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, "")
	checks, err := ParseFiles([]string{path})
	if err != nil {
		t.Fatalf("ParseFiles with empty file: %v", err)
	}
	if len(checks) != 0 {
		t.Fatalf("expected 0 checks for empty file, got %d", len(checks))
	}
}

func TestParseFilesEmptyList(t *testing.T) {
	t.Parallel()
	checks, err := ParseFiles([]string{})
	if err != nil {
		t.Fatalf("ParseFiles with empty list: %v", err)
	}
	if len(checks) != 0 {
		t.Fatalf("expected 0 checks for empty list, got %d", len(checks))
	}
}

func TestParseJUnitMissingFile(t *testing.T) {
	t.Parallel()
	_, err := ParseJUnit("/nonexistent/junit.xml")
	if err == nil {
		t.Fatal("expected error for missing JUnit file")
	}
}

func TestParseSARIFMissingFile(t *testing.T) {
	t.Parallel()
	_, err := ParseSARIF("/nonexistent/sarif.json")
	if err == nil {
		t.Fatal("expected error for missing SARIF file")
	}
}

func TestParseSARIFInvalidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_, err := ParseSARIF(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseJUnitInvalidXML(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, "<testsuite><notclosed>")
	_, err := ParseJUnit(path)
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}
