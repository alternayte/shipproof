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
