package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/verification"
)

// newCoverageWorkspace builds a repository with a started change, a requirement
// sidecar, a plan, and recorded proof results. It installs the RunOverrides
// seam so the command resolves this root from the working directory without
// an os.Chdir call. RunOverrides is shared process state, so the caller must
// not run this helper in parallel.
func newCoverageWorkspace(t *testing.T, changeID string) string {
	t.Helper()

	root := t.TempDir()
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	root = resolved

	if err := os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := "version: 1\nschema_version: \"0.1\"\nverification:\n  command: true\n"
	if err := os.WriteFile(filepath.Join(root, ".shipproof", "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(root, changeID+".md")
	if err := os.WriteFile(source, []byte("# "+changeID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := change.Start(root, changeID, source, "", 1); err != nil {
		t.Fatal(err)
	}

	sidecar := `{"schema_version":"0.1","change_id":"` + changeID + `","adopter":"native","requirements":[` +
		`{"id":"` + changeID + `-R1","statement":"First.","provenance":"observed"},` +
		`{"id":"` + changeID + `-R2","statement":"Second.","provenance":"observed"}]}`
	if err := os.WriteFile(filepath.Join(root, ".shipproof", "changes", changeID, "requirements.json"), []byte(sidecar+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := `{"schema_version":"0.1","change_id":"` + changeID + `","requirements":[` +
		`{"id":"` + changeID + `-R1","proof":[{"type":"command","target":"a","command":"true"}]},` +
		`{"id":"` + changeID + `-R2","proof":[{"type":"human","target":"b","human":true,"rationale":"A person reads it."}]}],"invariants":[]}`
	if err := os.WriteFile(verification.Path(root, changeID), []byte(plan+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	return root
}

func TestCoverageReportsProvenAndAwaitingHuman(t *testing.T) {
	newCoverageWorkspace(t, "SP-800")

	var stdout, stderr bytes.Buffer
	if code := runVerification([]string{"run", "SP-800", "--proofs-only"}, &stdout, &stderr); code != 0 {
		t.Fatalf("verification run exit = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runCoverage([]string{"SP-800", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("coverage exit = %d, stderr = %s", code, stderr.String())
	}

	var matrix coverage.Matrix
	if err := json.Unmarshal(stdout.Bytes(), &matrix); err != nil {
		t.Fatalf("parse coverage output: %v; output = %s", err, stdout.String())
	}
	if len(matrix.Rows) != 2 {
		t.Fatalf("Rows = %d, want 2", len(matrix.Rows))
	}
	if matrix.Rows[0].State != coverage.Proven {
		t.Fatalf("row 1 state = %q, want %q", matrix.Rows[0].State, coverage.Proven)
	}
	if matrix.Rows[1].State != coverage.AwaitingHuman {
		t.Fatalf("row 2 state = %q, want %q", matrix.Rows[1].State, coverage.AwaitingHuman)
	}
}

func TestCoverageNeverReportsInferredProvenance(t *testing.T) {
	newCoverageWorkspace(t, "SP-801")

	var stdout, stderr bytes.Buffer
	if code := runVerification([]string{"run", "SP-801", "--proofs-only"}, &stdout, &stderr); code != 0 {
		t.Fatalf("verification run exit = %d", code)
	}
	stdout.Reset()
	if code := runCoverage([]string{"SP-801", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("coverage exit = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "inferred") {
		t.Fatalf("coverage output holds inferred provenance: %s", stdout.String())
	}
}

func TestCoverageWithNoProofResultsReportsUnproven(t *testing.T) {
	newCoverageWorkspace(t, "SP-802")

	var stdout, stderr bytes.Buffer
	if code := runCoverage([]string{"SP-802", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("coverage exit = %d, stderr = %s", code, stderr.String())
	}
	var matrix coverage.Matrix
	if err := json.Unmarshal(stdout.Bytes(), &matrix); err != nil {
		t.Fatal(err)
	}
	if matrix.Rows[0].State != coverage.Unproven {
		t.Fatalf("row 1 state = %q, want %q", matrix.Rows[0].State, coverage.Unproven)
	}
	if matrix.RunCurrent {
		t.Fatal("RunCurrent = true with no recorded results")
	}
}

func TestCoverageTextFormNamesEveryRequirement(t *testing.T) {
	newCoverageWorkspace(t, "SP-803")

	var stdout, stderr bytes.Buffer
	if code := runCoverage([]string{"SP-803"}, &stdout, &stderr); code != 0 {
		t.Fatalf("coverage exit = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"SP-803-R1", "SP-803-R2", "unproven"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output has no %q: %s", want, stdout.String())
		}
	}
}

func TestCoverageWithNoRequirementSidecarFails(t *testing.T) {
	root := newCoverageWorkspace(t, "SP-804")
	if err := os.Remove(filepath.Join(root, ".shipproof", "changes", "SP-804", "requirements.json")); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := runCoverage([]string{"SP-804"}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "doc adopt") {
		t.Fatalf("stderr = %q, want the repair command", stderr.String())
	}
}

func TestCoverageWithNoArgumentIsAUsageError(t *testing.T) {
	newCoverageWorkspace(t, "SP-805")

	var stdout, stderr bytes.Buffer
	if code := runCoverage(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}
