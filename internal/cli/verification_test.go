package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
