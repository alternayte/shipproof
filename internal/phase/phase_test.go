package phase

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/change"
)

func TestResolveNoChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	result, err := Resolve(root, "SP-999")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NoChange {
		t.Fatalf("Phase = %q, want %q", result.Phase, NoChange)
	}
	if result.NextSkill != "prepare-change" {
		t.Fatalf("NextSkill = %q, want %q", result.NextSkill, "prepare-change")
	}
}

func TestResolveIntentStale(t *testing.T) {
	t.Parallel()

	root, source := newChange(t, "SP-300", 1)
	if err := os.WriteFile(source, []byte("# SP-300\n\nchanged after the snapshot\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve(root, "SP-300")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != IntentStale {
		t.Fatalf("Phase = %q, want %q", result.Phase, IntentStale)
	}
}

func TestResolveNeedsPlanWhenPlanIsAbsent(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-301", 1)

	result, err := Resolve(root, "SP-301")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NeedsPlan {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsPlan)
	}
	if result.NextCommand != "shipproof verification init SP-301" {
		t.Fatalf("NextCommand = %q", result.NextCommand)
	}
}

func TestResolveNeedsPlanWhenPlanIsEmpty(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-302", 1)
	writePlan(t, root, "SP-302", `{"schema_version":"0.1","change_id":"SP-302","requirements":[],"invariants":[]}`)

	result, err := Resolve(root, "SP-302")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NeedsPlan {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsPlan)
	}
}

func TestResolveLevelZeroSkipsNeedsPlan(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-303", 0)

	result, err := Resolve(root, "SP-303")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase == NeedsPlan {
		t.Fatal("a level-0 change must never reach NEEDS_PLAN")
	}
	if result.Phase != NeedsRun {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsRun)
	}
}

func TestResolveCorruptRecordIsAnError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, ".shipproof", "changes", "SP-304")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "change.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Resolve(root, "SP-304"); err == nil {
		t.Fatal("expected an error for a corrupt change record, not a phase")
	}
}

// newChange creates a repository root with one started change and returns the
// root and the absolute path of the source document.
func newChange(t *testing.T, changeID string, ceremony int) (string, string) {
	t.Helper()

	root := t.TempDir()
	source := filepath.Join(root, changeID+".md")
	if err := os.WriteFile(source, []byte("# "+changeID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := change.Start(root, changeID, source, "", ceremony); err != nil {
		t.Fatalf("change.Start() error = %v", err)
	}
	return root, source
}

func writePlan(t *testing.T, root, changeID, body string) {
	t.Helper()

	path := filepath.Join(root, ".shipproof", "changes", changeID, "verification.json")
	if err := os.WriteFile(path, []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveNeedsRun(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-310", 0)

	result, err := Resolve(root, "SP-310")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NeedsRun {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsRun)
	}
	if result.NextCommand != "shipproof verification run SP-310" {
		t.Fatalf("NextCommand = %q", result.NextCommand)
	}
}

func TestResolveRunFailed(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-311", 0)
	writeRun(t, root, "SP-311", `{"schema_version":"0.1","change_id":"SP-311","exit_code":1,"duration_ms":10,"stdout_path":"a","stderr_path":"b","timestamp":"2026-08-25T10:00:00Z"}`)

	result, err := Resolve(root, "SP-311")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != RunFailed {
		t.Fatalf("Phase = %q, want %q", result.Phase, RunFailed)
	}
}

func TestResolveRunStaleOnRevisionMismatch(t *testing.T) {
	root, _ := newChangeInRepo(t, "SP-312", 0)
	writeRun(t, root, "SP-312", `{"schema_version":"0.1","change_id":"SP-312","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","head_rev":"0000000000000000000000000000000000000000","tree_clean":true,"timestamp":"2026-08-25T10:00:00Z"}`)

	result, err := Resolve(root, "SP-312")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != RunStale {
		t.Fatalf("Phase = %q, want %q", result.Phase, RunStale)
	}
}

func TestResolveRunStaleBeatsRunFailed(t *testing.T) {
	root, _ := newChangeInRepo(t, "SP-313", 0)
	writeRun(t, root, "SP-313", `{"schema_version":"0.1","change_id":"SP-313","exit_code":1,"duration_ms":10,"stdout_path":"a","stderr_path":"b","head_rev":"0000000000000000000000000000000000000000","tree_clean":true,"timestamp":"2026-08-25T10:00:00Z"}`)

	result, err := Resolve(root, "SP-313")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != RunStale {
		t.Fatalf("Phase = %q, want %q; a stale exit code must not be reported", result.Phase, RunStale)
	}
}

func TestResolveRunStaleOnDirtyTree(t *testing.T) {
	root, _ := newChangeInRepo(t, "SP-319", 0)
	head := headRevision(t, root)
	writeRun(t, root, "SP-319", `{"schema_version":"0.1","change_id":"SP-319","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","head_rev":"`+head+`","tree_clean":true,"timestamp":"2026-08-25T10:00:00Z"}`)

	// Dirty a file that is not the change source. Editing the source document
	// would make the intent stale, and Resolve reports that before any run phase.
	if err := os.WriteFile(filepath.Join(root, "other.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve(root, "SP-319")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != RunStale {
		t.Fatalf("Phase = %q, want %q", result.Phase, RunStale)
	}
}

func TestResolveRunStaleWhenTheRunItselfWasDirty(t *testing.T) {
	root, _ := newChangeInRepo(t, "SP-320", 0)
	head := headRevision(t, root)
	writeRun(t, root, "SP-320", `{"schema_version":"0.1","change_id":"SP-320","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","head_rev":"`+head+`","tree_clean":false,"timestamp":"2026-08-25T10:00:00Z"}`)

	result, err := Resolve(root, "SP-320")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != RunStale {
		t.Fatalf("Phase = %q, want %q; a run of a dirty tree proves nothing", result.Phase, RunStale)
	}
}

// TestResolveShipproofWriteDoesNotStaleTheRun proves criterion S3b. Writing the
// evidence pack must advance the phase, never pin it at RUN_STALE.
func TestResolveShipproofWriteDoesNotStaleTheRun(t *testing.T) {
	root, _ := newChangeInRepo(t, "SP-321", 0)
	head := headRevision(t, root)
	writeRun(t, root, "SP-321", `{"schema_version":"0.1","change_id":"SP-321","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","head_rev":"`+head+`","tree_clean":true,"timestamp":"2026-08-25T10:00:00Z"}`)
	writeArtifact(t, root, "SP-321", "evidence-pack.json", "{}")

	result, err := Resolve(root, "SP-321")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != ReadyForHuman {
		t.Fatalf("Phase = %q, want %q; a ShipProof write must not stale the run", result.Phase, ReadyForHuman)
	}
}

func TestResolveRunWithNoRevisionIsNeverStale(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-314", 0)
	writeRun(t, root, "SP-314", `{"schema_version":"0.1","change_id":"SP-314","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","timestamp":"2026-08-25T10:00:00Z"}`)

	result, err := Resolve(root, "SP-314")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NeedsEvidence {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsEvidence)
	}
}

func TestResolveNeedsReviewPacket(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-315", 1)
	writePlan(t, root, "SP-315", `{"schema_version":"0.1","change_id":"SP-315","requirements":[{"id":"SP-315-R1","proof":[{"type":"unit","target":"x_test.go","command":"go test ."}]}],"invariants":[]}`)
	writeRun(t, root, "SP-315", `{"schema_version":"0.1","change_id":"SP-315","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","timestamp":"2026-08-25T10:00:00Z"}`)
	writeArtifact(t, root, "SP-315", "evidence-pack.json", "{}")

	result, err := Resolve(root, "SP-315")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NeedsReviewPacket {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsReviewPacket)
	}
}

func TestResolveLevelZeroSkipsReviewPacket(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-316", 0)
	writeRun(t, root, "SP-316", `{"schema_version":"0.1","change_id":"SP-316","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","timestamp":"2026-08-25T10:00:00Z"}`)
	writeArtifact(t, root, "SP-316", "evidence-pack.json", "{}")

	result, err := Resolve(root, "SP-316")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != ReadyForHuman {
		t.Fatalf("Phase = %q, want %q", result.Phase, ReadyForHuman)
	}
}

func TestResolveReadyForHuman(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-317", 1)
	writePlan(t, root, "SP-317", `{"schema_version":"0.1","change_id":"SP-317","requirements":[{"id":"SP-317-R1","proof":[{"type":"unit","target":"x_test.go","command":"go test ."}]}],"invariants":[]}`)
	writeRun(t, root, "SP-317", `{"schema_version":"0.1","change_id":"SP-317","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","timestamp":"2026-08-25T10:00:00Z"}`)
	writeArtifact(t, root, "SP-317", "evidence-pack.json", "{}")
	writeArtifact(t, root, "SP-317", "review-packet.json", "{}")

	result, err := Resolve(root, "SP-317")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != ReadyForHuman {
		t.Fatalf("Phase = %q, want %q", result.Phase, ReadyForHuman)
	}
}

// TestResolveStoresNoCursor proves the two properties that the design rests on.
// Repeated calls agree, and the removal of an artifact moves the phase backward.
func TestResolveStoresNoCursor(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-318", 0)
	writeRun(t, root, "SP-318", `{"schema_version":"0.1","change_id":"SP-318","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","timestamp":"2026-08-25T10:00:00Z"}`)
	writeArtifact(t, root, "SP-318", "evidence-pack.json", "{}")

	first, err := Resolve(root, "SP-318")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	second, err := Resolve(root, "SP-318")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if first != second {
		t.Fatalf("two calls disagree: %+v and %+v", first, second)
	}
	if first.Phase != ReadyForHuman {
		t.Fatalf("Phase = %q, want %q", first.Phase, ReadyForHuman)
	}

	packPath := filepath.Join(root, ".shipproof", "changes", "SP-318", "evidence-pack.json")
	if err := os.Remove(packPath); err != nil {
		t.Fatal(err)
	}

	third, err := Resolve(root, "SP-318")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if third.Phase != NeedsEvidence {
		t.Fatalf("Phase = %q after artifact removal, want %q", third.Phase, NeedsEvidence)
	}
}

func writeRun(t *testing.T, root, changeID, body string) {
	t.Helper()

	dir := filepath.Join(root, ".shipproof", "runs", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeArtifact(t *testing.T, root, changeID, name, body string) {
	t.Helper()

	path := filepath.Join(root, ".shipproof", "changes", changeID, name)
	if err := os.WriteFile(path, []byte(body+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func headRevision(t *testing.T, root string) string {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// newChangeInRepo is newChange inside a real Git repository, so the staleness
// rule has a HEAD to compare against. The change record lands under
// .shipproof/, which the tree check excludes, so the fixture stays clean.
func newChangeInRepo(t *testing.T, changeID string, ceremony int) (string, string) {
	t.Helper()

	root := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	source := filepath.Join(root, changeID+".md")
	if err := os.WriteFile(source, []byte("# "+changeID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if _, err := change.Start(root, changeID, source, "", ceremony); err != nil {
		t.Fatalf("change.Start() error = %v", err)
	}
	return root, source
}

func TestResolveIntentStaleNamesAWorkingCommand(t *testing.T) {
	t.Parallel()

	root, source := newChange(t, "SP-320", 2)
	if err := os.WriteFile(source, []byte("# SP-320\n\nchanged after the snapshot\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Resolve(root, "SP-320")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != IntentStale {
		t.Fatalf("Phase = %q, want %q", result.Phase, IntentStale)
	}
	if !strings.HasSuffix(result.NextCommand, "--force") {
		t.Fatalf("NextCommand = %q, want a --force suffix", result.NextCommand)
	}

	// The named command must succeed, or the phase traps the change.
	fields := strings.Fields(result.NextCommand)
	sourceArgument := filepath.Join(root, filepath.FromSlash(fields[len(fields)-2]))
	if _, err := change.Restart(root, "SP-320", sourceArgument, "", nil); err != nil {
		t.Fatalf("the named command failed: %v", err)
	}

	result, err = Resolve(root, "SP-320")
	if err != nil {
		t.Fatalf("Resolve() after the named command error = %v", err)
	}
	if result.Phase == IntentStale {
		t.Fatal("the named command left the change at INTENT_STALE")
	}
}

func TestResolveNeedsPlanOnEmptyPlanNamesVerificationCheck(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-321", 1)
	writePlan(t, root, "SP-321", `{"schema_version":"0.1","change_id":"SP-321","requirements":[],"invariants":[]}`)

	result, err := Resolve(root, "SP-321")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NeedsPlan {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsPlan)
	}
	if result.NextCommand != "shipproof verification check SP-321" {
		t.Fatalf("NextCommand = %q, want %q", result.NextCommand, "shipproof verification check SP-321")
	}
}

func TestResolveNeedsEvidence(t *testing.T) {
	t.Parallel()

	root, _ := newChange(t, "SP-318", 1)
	writePlan(t, root, "SP-318", `{"schema_version":"0.1","change_id":"SP-318","requirements":[{"id":"SP-318-R1","proof":[{"type":"unit","target":"x_test.go","command":"go test ."}]}],"invariants":[]}`)
	writeRun(t, root, "SP-318", `{"schema_version":"0.1","change_id":"SP-318","exit_code":0,"duration_ms":10,"stdout_path":"a","stderr_path":"b","timestamp":"2026-08-25T10:00:00Z"}`)

	result, err := Resolve(root, "SP-318")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.Phase != NeedsEvidence {
		t.Fatalf("Phase = %q, want %q", result.Phase, NeedsEvidence)
	}
	if result.NextSkill != "produce-evidence" {
		t.Fatalf("NextSkill = %q, want produce-evidence", result.NextSkill)
	}
	if result.NextCommand != "shipproof evidence pack SP-318" {
		t.Fatalf("NextCommand = %q", result.NextCommand)
	}
}
