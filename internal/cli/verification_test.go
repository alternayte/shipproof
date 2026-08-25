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

const tiePlanJSON = `{"schema_version":"0.1","change_id":"SP-028","requirements":[{"id":"SP-028-R1","proof":[{"type":"unit","target":"x_test.go"}]}],"invariants":[]}`

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

// runVerificationIn runs the verification command with the process working
// directory set to root. The check resolves the repository root from ".", so
// the test must move there.
func runVerificationIn(t *testing.T, root string, args []string, stdout, stderr *bytes.Buffer) int {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	})
	return runVerification(args, stdout, stderr)
}
