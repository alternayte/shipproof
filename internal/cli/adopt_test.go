package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const nativeDoc = `# SP-011 — Cycle time and rework metrics

## Requirements

### SP-011-R1 — Derive per-change cycle time

Body.

### SP-011-R2 — Derive project average cycle time

Body.
`

const foreignDoc = `# Payment retry specification

## Retry a failed charge

Body.

## Cap the retry count

- MUST stop after five attempts.
`

// adoptRepo creates a repository root with a ShipProof state directory and one
// source document, then returns the root and the document path.
func adoptRepo(t *testing.T, name, body string) (string, string) {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755); err != nil {
		t.Fatal(err)
	}
	docs := filepath.Join(root, "docs", "changes")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(docs, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
	return root, path
}

func TestDocAdoptNativeWritesTheSidecar(t *testing.T) {
	root, source := adoptRepo(t, "SP-011.md", nativeDoc)

	var stdout, stderr bytes.Buffer
	code := runDoc([]string{"adopt", "SP-011", "--source", source}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "changes", "SP-011", "requirements.json"))
	if err != nil {
		t.Fatalf("sidecar not written: %v", err)
	}
	var set struct {
		Adopter      string `json:"adopter"`
		Requirements []struct {
			ID         string `json:"id"`
			Provenance string `json:"provenance"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(data, &set); err != nil {
		t.Fatal(err)
	}
	if set.Adopter != "native" {
		t.Fatalf("adopter = %q, want native", set.Adopter)
	}
	if len(set.Requirements) != 2 {
		t.Fatalf("requirements = %d, want 2", len(set.Requirements))
	}
	for _, requirement := range set.Requirements {
		if requirement.Provenance != "observed" {
			t.Fatalf("%s provenance = %q, want observed", requirement.ID, requirement.Provenance)
		}
	}
	if !strings.Contains(stdout.String(), "2 requirements") {
		t.Fatalf("stdout does not report the count:\n%s", stdout.String())
	}
}

func TestDocAdoptForeignRefusesWithoutConfirm(t *testing.T) {
	root, source := adoptRepo(t, "spec.md", foreignDoc)

	var stdout, stderr bytes.Buffer
	code := runDoc([]string{"adopt", "SP-050", "--source", source}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero\nstdout: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".shipproof", "changes", "SP-050", "requirements.json")); err == nil {
		t.Fatal("the sidecar was written without a confirmation")
	}
	if !strings.Contains(stdout.String(), "Retry a failed charge") {
		t.Fatalf("stdout does not present the proposal:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--confirm") {
		t.Fatalf("stderr does not name --confirm:\n%s", stderr.String())
	}
}

func TestDocAdoptForeignWritesWithConfirm(t *testing.T) {
	root, source := adoptRepo(t, "spec.md", foreignDoc)

	var stdout, stderr bytes.Buffer
	code := runDoc([]string{"adopt", "SP-050", "--source", source, "--confirm"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "changes", "SP-050", "requirements.json"))
	if err != nil {
		t.Fatalf("sidecar not written: %v", err)
	}
	var set struct {
		Adopter      string `json:"adopter"`
		Requirements []struct {
			Provenance  string `json:"provenance"`
			ConfirmedAt string `json:"confirmed_at"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(data, &set); err != nil {
		t.Fatal(err)
	}
	if set.Adopter != "foreign" {
		t.Fatalf("adopter = %q, want foreign", set.Adopter)
	}
	for index, requirement := range set.Requirements {
		if requirement.Provenance != "human" {
			t.Fatalf("requirement %d provenance = %q, want human", index+1, requirement.Provenance)
		}
		if requirement.ConfirmedAt == "" {
			t.Fatalf("requirement %d carries no confirmation stamp", index+1)
		}
	}
}

func TestDocAdoptJSONOutput(t *testing.T) {
	_, source := adoptRepo(t, "SP-011.md", nativeDoc)

	var stdout, stderr bytes.Buffer
	code := runDoc([]string{"adopt", "SP-011", "--source", source, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", code, stderr.String())
	}
	var set struct {
		SchemaVersion string `json:"schema_version"`
		ChangeID      string `json:"change_id"`
		Adopter       string `json:"adopter"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &set); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if set.ChangeID != "SP-011" || set.Adopter != "native" || set.SchemaVersion != "0.1" {
		t.Fatalf("unexpected object: %+v", set)
	}
}

func TestDocAdoptRejectsAMissingSource(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runDoc([]string{"adopt", "SP-011"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestDocAdoptRejectsAnUnknownOption(t *testing.T) {
	_, source := adoptRepo(t, "SP-011.md", nativeDoc)

	var stdout, stderr bytes.Buffer
	code := runDoc([]string{"adopt", "SP-011", "--source", source, "--unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestDocAdoptRejectsAnOptionAsTheSourceValue(t *testing.T) {
	adoptRepo(t, "SP-011.md", nativeDoc)

	var stdout, stderr bytes.Buffer
	code := runDoc([]string{"adopt", "SP-011", "--source", "--confirm"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr does not report a usage error:\n%s", stderr.String())
	}
}

func TestDocAdoptRefusesToOverwriteWithoutForce(t *testing.T) {
	root, source := adoptRepo(t, "SP-011.md", nativeDoc)
	sidecar := filepath.Join(root, ".shipproof", "changes", "SP-011", "requirements.json")

	var stdout, stderr bytes.Buffer
	if code := runDoc([]string{"adopt", "SP-011", "--source", source}, &stdout, &stderr); code != 0 {
		t.Fatalf("first adopt exit = %d, want 0\nstderr: %s", code, stderr.String())
	}
	before, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code := runDoc([]string{"adopt", "SP-011", "--source", source}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("second adopt exit = 0, want non-zero\nstdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--force") {
		t.Fatalf("stderr does not name --force:\n%s", stderr.String())
	}
	after, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("the refused adopt changed the existing sidecar")
	}
}

func TestDocAdoptOverwritesWithForce(t *testing.T) {
	root, source := adoptRepo(t, "SP-011.md", nativeDoc)
	sidecar := filepath.Join(root, ".shipproof", "changes", "SP-011", "requirements.json")

	var stdout, stderr bytes.Buffer
	if code := runDoc([]string{"adopt", "SP-011", "--source", source}, &stdout, &stderr); code != 0 {
		t.Fatalf("first adopt exit = %d, want 0\nstderr: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runDoc([]string{"adopt", "SP-011", "--source", source, "--force"}, &stdout, &stderr); code != 0 {
		t.Fatalf("forced adopt exit = %d, want 0\nstderr: %s", code, stderr.String())
	}
	if _, err := os.ReadFile(sidecar); err != nil {
		t.Fatalf("sidecar missing after the forced adopt: %v", err)
	}
}

func TestDocAdoptRejectsAMissingChangeID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runDoc([]string{"adopt"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestDocAdoptReportsAnAbsentSourceFile(t *testing.T) {
	root, _ := adoptRepo(t, "SP-011.md", nativeDoc)

	var stdout, stderr bytes.Buffer
	code := runDoc([]string{"adopt", "SP-011", "--source", filepath.Join(root, "absent.md")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}
