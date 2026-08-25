package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/proofs"
	"github.com/alternayte/shipproof/internal/verification"
)

// tieRepo writes a change directory holding a verification plan and, when
// sidecar is non-empty, a requirement sidecar.
func tieRepo(t *testing.T, plan, sidecar string) string {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, ".shipproof", "changes", "SP-028")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "verification.json"), []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	if sidecar != "" {
		if err := os.WriteFile(filepath.Join(dir, "requirements.json"), []byte(sidecar), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const tiePlanJSON = `{"schema_version":"0.1","change_id":"SP-028","requirements":[{"id":"SP-028-R1","proof":[{"type":"unit","target":"x_test.go","command":"go test ."}]}],"invariants":[]}`

const tieSidecarMatching = `{"schema_version":"0.1","change_id":"SP-028","adopter":"native","requirements":[{"id":"SP-028-R1","statement":"A","provenance":"observed"}]}`

const tieSidecarExtra = `{"schema_version":"0.1","change_id":"SP-028","adopter":"native","requirements":[{"id":"SP-028-R1","statement":"A","provenance":"observed"},{"id":"SP-028-R2","statement":"B","provenance":"observed"}]}`

const tieSidecarMissing = `{"schema_version":"0.1","change_id":"SP-028","adopter":"native","requirements":[{"id":"SP-028-R7","statement":"A","provenance":"observed"}]}`

func TestVerificationCheckPassesWithNoSidecar(t *testing.T) {
	root := tieRepo(t, tiePlanJSON, "")

	var stdout, stderr bytes.Buffer
	code := runVerificationIn(t, root, []string{"check", "SP-028"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", code, stderr.String())
	}
}

func TestVerificationCheckPassesWhenTheSetsMatch(t *testing.T) {
	root := tieRepo(t, tiePlanJSON, tieSidecarMatching)

	var stdout, stderr bytes.Buffer
	code := runVerificationIn(t, root, []string{"check", "SP-028"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "tie check") {
		t.Fatalf("stdout does not report the tie check:\n%s", stdout.String())
	}
}

func TestVerificationCheckBlocksAnUnplannedRequirement(t *testing.T) {
	root := tieRepo(t, tiePlanJSON, tieSidecarExtra)

	var stdout, stderr bytes.Buffer
	code := runVerificationIn(t, root, []string{"check", "SP-028"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero\nstdout: %s", stdout.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "BLOCKER") {
		t.Fatalf("output does not report a blocker:\n%s", combined)
	}
	if !strings.Contains(combined, "SP-028-R2") {
		t.Fatalf("output does not name SP-028-R2:\n%s", combined)
	}
}

func TestVerificationCheckBlocksAnUntiedPlanEntry(t *testing.T) {
	root := tieRepo(t, tiePlanJSON, tieSidecarMissing)

	var stdout, stderr bytes.Buffer
	code := runVerificationIn(t, root, []string{"check", "SP-028"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero\nstdout: %s", stdout.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "SP-028-R1") {
		t.Fatalf("output does not name the untied plan entry:\n%s", combined)
	}
	if !strings.Contains(combined, "SP-028-R7") {
		t.Fatalf("output does not name the unplanned requirement:\n%s", combined)
	}
}

// runVerificationIn runs the verification command with the repository root
// resolved to root. The check resolves the root from ".", so the test installs
// the RunOverrides seam that the rest of the package uses.
func runVerificationIn(t *testing.T, root string, args []string, stdout, stderr *bytes.Buffer) int {
	t.Helper()

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
	return runVerification(args, stdout, stderr)
}

// TestVerificationCheckTiesAPlanFileByPath proves that an existing plan file
// locates its own repository. No override is installed, so
// a resolution from "." finds the ShipProof repository that holds this test
// and never finds the temporary sidecar.
func TestVerificationCheckTiesAPlanFileByPath(t *testing.T) {
	root := tieRepo(t, tiePlanJSON, tieSidecarExtra)
	plan := filepath.Join(root, ".shipproof", "changes", "SP-028", "verification.json")

	var stdout, stderr bytes.Buffer
	code := runVerification([]string{"check", plan}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("exit = 0, want non-zero\nstdout: %s", stdout.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "SP-028-R2") {
		t.Fatalf("output does not report the tie blocker:\n%s", combined)
	}
}

func TestVerificationRunWritesProofResults(t *testing.T) {
	root := newVerificationRunWorkspace(t, "SP-700")

	var stdout, stderr bytes.Buffer
	if code := runVerification([]string{"run", "SP-700"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}

	set, err := proofs.Load(root, "SP-700")
	if err != nil {
		t.Fatalf("proofs.Load() error = %v", err)
	}
	if len(set.Results) != 2 {
		t.Fatalf("Results = %d, want 2", len(set.Results))
	}
	if set.Results[0].Status != proofs.Pass {
		t.Fatalf("result 1 status = %q", set.Results[0].Status)
	}
	if set.Results[1].Status != proofs.Fail {
		t.Fatalf("result 2 status = %q", set.Results[1].Status)
	}
}

func TestVerificationRunGateOnlyWritesNoProofResults(t *testing.T) {
	root := newVerificationRunWorkspace(t, "SP-701")

	var stdout, stderr bytes.Buffer
	if code := runVerification([]string{"run", "SP-701", "--gate-only"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if proofs.Exists(root, "SP-701") {
		t.Fatal("--gate-only wrote proofs.json")
	}
	if _, err := os.Stat(filepath.Join(root, ".shipproof", "runs", "SP-701", "run.json")); err != nil {
		t.Fatalf("--gate-only wrote no run.json: %v", err)
	}
}

func TestVerificationRunProofsOnlyWritesNoRunRecord(t *testing.T) {
	root := newVerificationRunWorkspace(t, "SP-702")

	var stdout, stderr bytes.Buffer
	if code := runVerification([]string{"run", "SP-702", "--proofs-only"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !proofs.Exists(root, "SP-702") {
		t.Fatal("--proofs-only wrote no proofs.json")
	}
	if _, err := os.Stat(filepath.Join(root, ".shipproof", "runs", "SP-702", "run.json")); err == nil {
		t.Fatal("--proofs-only wrote run.json")
	}
}

func TestVerificationRunRejectsBothScopeFlags(t *testing.T) {
	newVerificationRunWorkspace(t, "SP-703")

	var stdout, stderr bytes.Buffer
	if code := runVerification([]string{"run", "SP-703", "--gate-only", "--proofs-only"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestVerificationRunWithNoPlanStatesWhyItRanNoProof(t *testing.T) {
	root := newVerificationRunWorkspace(t, "SP-704")
	if err := os.Remove(verification.Path(root, "SP-704")); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := runVerification([]string{"run", "SP-704"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no verification plan") {
		t.Fatalf("stdout = %q, want a stated reason", stdout.String())
	}
	if proofs.Exists(root, "SP-704") {
		t.Fatal("a change with no plan wrote proofs.json")
	}
}

// newVerificationRunWorkspace builds a repository with one started change, a
// gate command that passes, and a plan with one passing and one failing proof.
// It installs the RunOverrides seam so the command resolves this root from
// the working directory without an os.Chdir call, because RunOverrides is
// shared process state and the caller must not run this helper in parallel.
func newVerificationRunWorkspace(t *testing.T, changeID string) string {
	t.Helper()

	root := t.TempDir()
	// t.TempDir can return a symlinked path on macOS. Resolve it so that the
	// root the CLI finds matches the root the test asserts on.
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

	plan := `{"schema_version":"0.1","change_id":"` + changeID + `","requirements":[{"id":"` + changeID + `-R1","proof":[{"type":"command","target":"a","command":"true"},{"type":"command","target":"b","command":"exit 3"}]}],"invariants":[]}`
	if err := os.WriteFile(verification.Path(root, changeID), []byte(plan+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	return root
}

// writeConfigWithCoverage overwrites config.yaml with a coverage command that
// writes a fixed profile to {{profile}}. The caller supplies a root that
// newVerificationRunWorkspace already built.
func writeConfigWithCoverage(t *testing.T, root string) {
	t.Helper()

	config := "version: 1\nschema_version: \"0.1\"\nverification:\n  command: true\n  coverage:\n    command: printf 'mode: set\\nm/internal/a/a.go:1.1,2.2 1 1\\n' > {{profile}}\n"
	if err := os.WriteFile(filepath.Join(root, ".shipproof", "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVerificationRunWritesAMergedProfile(t *testing.T) {
	root := newVerificationRunWorkspace(t, "SP-200")
	writeConfigWithCoverage(t, root)

	var stdout, stderr bytes.Buffer
	code := runVerification([]string{"run", "SP-200", "--proofs-only"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "coverage:") {
		t.Errorf("stdout names no merged profile: %s", stdout.String())
	}
	if _, err := os.Stat(proofs.MergedProfilePath(root, "SP-200")); err != nil {
		t.Errorf("merged profile missing: %v", err)
	}
}

// TestVerificationRunSurvivesAFailedCoverageCleanup proves that a coverage
// directory ShipProof cannot clear does not block the run and does not
// change the exit code. SDD Section 11.6: the signal never blocks a run.
//
// The test locks a subdirectory nested inside the coverage directory, not
// the coverage directory's own parent. The parent stays writable, so
// proofs.json and run.json still write next to it. os.RemoveAll cannot
// unlink an entry inside the locked subdirectory, so it fails and leaves a
// stale profile behind.
func TestVerificationRunSurvivesAFailedCoverageCleanup(t *testing.T) {
	root := newVerificationRunWorkspace(t, "SP-201")
	writeConfigWithCoverage(t, root)

	coverageDir := proofs.CoverageDir(root, "SP-201")
	locked := filepath.Join(coverageDir, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(locked, "stale.out")
	if err := os.WriteFile(stale, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	var stdout, stderr bytes.Buffer
	code := runVerification([]string{"run", "SP-201", "--proofs-only"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("a failed coverage cleanup changed the exit code: %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "coverage") {
		t.Errorf("stderr does not report the cleanup failure: %s", stderr.String())
	}
	if _, err := os.Stat(stale); err != nil {
		t.Errorf("the stale profile was removed despite the failed cleanup: %v", err)
	}
	if !strings.Contains(stdout.String(), "stale profile") {
		t.Errorf("the coverage summary does not warn about a stale profile: %s", stdout.String())
	}
}
