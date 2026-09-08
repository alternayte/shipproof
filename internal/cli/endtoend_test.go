package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/proofs"
	"github.com/alternayte/shipproof/internal/schema"
)

// initRepository makes the test root a git repository with one commit.
// ShipProof compares a recorded result against the head revision, so a
// directory with no history can never hold a current result. That refusal is
// correct, and this helper gives the test the history it needs.
func initRepository(t *testing.T, root string) {
	t.Helper()
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"add", "-A"},
		{"commit", "-qm", "initial"},
	} {
		command := exec.Command("git", arguments...)
		command.Dir = root
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, out)
		}
	}
}

// writePlanWithProof writes the verification plan that an agent fills in
// between `start` and `pack`. The plan is the agent's work, not a command.
func writePlanWithProof(t *testing.T, root, changeID, requirementID, command string) {
	t.Helper()
	plan := map[string]any{
		"schema_version": "0.1",
		"change_id":      changeID,
		"requirements": []map[string]any{{
			"id": requirementID,
			"proof": []map[string]any{{
				"type": "command", "target": command, "command": command,
			}},
		}},
		"invariants": []map[string]any{},
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".shipproof", "changes", changeID, "verification.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTwoCommandsReachAFullResult is the proof for row C5. Section 4 says that
// `pack` runs `prove` when no fresh result exists, so `start` and then `pack`
// reach a full result from a clean repository.
func TestTwoCommandsReachAFullResult(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "intent.md",
		"# SP-600\n\n### SP-600-R1 — The tool answers\n\nThe command exits zero.\n")

	// Command one.
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-600", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("start: exit code = %d, stderr = %s", code, stderr.String())
	}
	writePlanWithProof(t, root, "SP-600", "SP-600-R1", "true")
	initRepository(t, root)

	// Command two. Nothing ran `prove`, so `pack` must run it.
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"pack", "SP-600"}, &stdout, &stderr); code != 0 {
		t.Fatalf("pack: exit code = %d, stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}

	// `pack` must have produced the proof result itself.
	if !proofs.Exists(root, "SP-600") {
		t.Fatalf("pack recorded no proof result:\n%s", stdout.String())
	}
	recorded, err := proofs.Load(root, "SP-600")
	if err != nil {
		t.Fatal(err)
	}
	if len(recorded.Results) != 1 {
		t.Fatalf("pack recorded %d proof results, want 1", len(recorded.Results))
	}

	pack := readPack(t, root, "SP-600")
	if pack.Verdict.Verdict == "" {
		t.Fatal("the pack states no verdict")
	}
	if len(pack.Requirements) != 1 {
		t.Fatalf("the pack holds %d requirement rows, want 1", len(pack.Requirements))
	}
	if pack.Requirements[0].State != "proven" {
		t.Fatalf("the requirement reads %q, want proven: %+v", pack.Requirements[0].State, pack.Requirements[0])
	}
	if len(pack.Checks) == 0 {
		t.Fatal("the pack holds no check")
	}
}

// TestAnUnplannedRequirementStillProducesACompletePack holds rule 1 of Section
// 7. A failing `prove` is a state that the pack reports, not a reason to
// produce nothing.
func TestAnUnplannedRequirementStillProducesACompletePack(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "intent.md",
		"# SP-602\n\n### SP-602-R1 — The tool answers\n\nThe command exits zero.\n")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-602", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("start: exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"pack", "SP-602"}, &stdout, &stderr); code != 0 {
		t.Fatalf("pack: exit code = %d, stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}

	pack := readPack(t, root, "SP-602")
	if pack.Verdict.Verdict != "NOT PROVEN" {
		t.Fatalf("verdict = %q, want NOT PROVEN", pack.Verdict.Verdict)
	}
	if len(pack.Requirements) != 1 {
		t.Fatalf("the pack holds %d requirement rows, want 1", len(pack.Requirements))
	}
	if pack.Requirements[0].State == "proven" {
		t.Fatal("an unplanned requirement reads proven")
	}
}

func readPack(t *testing.T, root, changeID string) schema.EvidencePack {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "changes", changeID, "evidence-pack.json"))
	if err != nil {
		t.Fatalf("read the pack: %v", err)
	}
	var pack schema.EvidencePack
	if err := json.Unmarshal(data, &pack); err != nil {
		t.Fatal(err)
	}
	return pack
}

// TestPackSkipsProveOnRequest holds requirement R7. A caller that already ran
// `prove` must be able to say so.
func TestPackSkipsProveOnRequest(t *testing.T) {
	root := newCLITestRoot(t)
	source := writeIntent(t, root, "intent.md",
		"# SP-601\n\n### SP-601-R1 — The tool answers\n\nThe command exits zero.\n")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"start", "SP-601", "--intent", source, "--ceremony", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("start: exit code = %d, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"pack", "SP-601", "--no-prove"}, &stdout, &stderr); code != 0 {
		t.Fatalf("pack: exit code = %d, stderr = %s", code, stderr.String())
	}
	if proofs.Exists(root, "SP-601") {
		t.Fatal("--no-prove ran the proofs anyway")
	}
	if strings.Contains(stdout.String(), "Proofs:") {
		t.Fatalf("--no-prove reported a proof run:\n%s", stdout.String())
	}
}
