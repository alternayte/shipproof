package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/requirements"
)

func startWith(t *testing.T, root, changeID, body string, extra ...string) (string, string) {
	t.Helper()
	source := writeIntent(t, root, "spec.md", body)
	args := append([]string{"start", changeID, "--intent", source, "--ceremony", "0"}, extra...)
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("start: exit %d: %s", code, stderr.String())
	}
	return stdout.String(), stderr.String()
}

// TestForceMergesANewRequirement holds question 2 of SP-040. This was a
// correctness bug: --force re-snapshotted the document and left the
// requirement set stale, so a pack reported a fresh intent against
// requirements the document no longer matched.
func TestForceMergesANewRequirement(t *testing.T) {
	root := newCLITestRoot(t)
	startWith(t, root, "SP-1", "# Feature\n\n- MUST do the first thing.\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"start", "SP-1", "--confirm-requirements"}, &out, &errOut); code != 0 {
		t.Fatalf("confirm: %s", errOut.String())
	}
	first, err := requirements.Load(root, "SP-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Requirements) != 1 {
		t.Fatalf("the set holds %d requirements, want 1", len(first.Requirements))
	}
	confirmedAt := first.Requirements[0].ConfirmedAt

	stdout, _ := startWith(t, root, "SP-1",
		"# Feature\n\n- MUST do the first thing.\n- MUST do the second thing.\n", "--force")

	second, err := requirements.Load(root, "SP-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Requirements) != 2 {
		t.Fatalf("the set holds %d requirements after --force, want 2:\n%s",
			len(second.Requirements), stdout)
	}
	// The requirement that already stood keeps its confirmation.
	if second.Requirements[0].ConfirmedAt != confirmedAt {
		t.Errorf("the merge re-stamped a requirement a person had already confirmed")
	}
	if !strings.Contains(stdout, "added") {
		t.Errorf("start does not report what it added:\n%s", stdout)
	}
}

// TestTheMergeNeverDeletes holds principle 6 of Section 2. A requirement the
// document no longer states is reported and left in place.
func TestTheMergeNeverDeletes(t *testing.T) {
	root := newCLITestRoot(t)
	startWith(t, root, "SP-2", "# Feature\n\n- MUST keep this.\n- MUST drop this.\n")
	var out, errOut bytes.Buffer
	if code := Run([]string{"start", "SP-2", "--confirm-requirements"}, &out, &errOut); code != 0 {
		t.Fatalf("confirm: %s", errOut.String())
	}

	stdout, _ := startWith(t, root, "SP-2", "# Feature\n\n- MUST keep this.\n", "--force")

	set, err := requirements.Load(root, "SP-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Requirements) != 2 {
		t.Fatalf("the merge deleted a requirement: the set holds %d, want 2", len(set.Requirements))
	}
	if !strings.Contains(stdout, "no longer") {
		t.Errorf("start does not report the requirement the document dropped:\n%s", stdout)
	}
}

// TestStatusNamesTheRequirementsWithNoProof holds question 4 of SP-040.
func TestStatusNamesTheRequirementsWithNoProof(t *testing.T) {
	root := newCLITestRoot(t)
	startWith(t, root, "SP-3", "# Feature\n\n- MUST do the thing.\n")
	var out, errOut bytes.Buffer
	if code := Run([]string{"start", "SP-3", "--confirm-requirements"}, &out, &errOut); code != 0 {
		t.Fatalf("confirm: %s", errOut.String())
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"status", "SP-3"}, &stdout, &stderr); code != 0 {
		t.Fatalf("status: exit %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SP-3-R1") {
		t.Errorf("status does not name the requirement that needs a proof:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "verification.json") {
		t.Errorf("status does not name the file to edit:\n%s", stdout.String())
	}
}

// TestStatusNamesAProofItCannotRun holds question 3 of SP-040. A command whose
// program is missing is a broken proof, not a failed requirement, and the two
// must never read the same.
func TestStatusNamesAProofItCannotRun(t *testing.T) {
	root := newCLITestRoot(t)
	startWith(t, root, "SP-4", "# Feature\n\n- MUST do the thing.\n")
	var out, errOut bytes.Buffer
	if code := Run([]string{"start", "SP-4", "--confirm-requirements"}, &out, &errOut); code != 0 {
		t.Fatalf("confirm: %s", errOut.String())
	}
	writePlanWithProof(t, root, "SP-4", "SP-4-R1", "definitely-not-a-real-program --run")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"status", "SP-4"}, &stdout, &stderr); code != 0 {
		t.Fatalf("status: exit %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "definitely-not-a-real-program") {
		t.Errorf("status does not name the program it cannot find:\n%s", stdout.String())
	}
}
