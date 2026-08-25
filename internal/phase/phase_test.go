package phase

import (
	"os"
	"path/filepath"
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
