package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsCurrentCannotJudgeARunWithNoRevision(t *testing.T) {
	t.Parallel()

	current, reason := IsCurrent(t.TempDir(), Result{})
	if !current {
		t.Fatalf("IsCurrent() = false, %q; an unjudgeable run must not be a false alarm", reason)
	}
	if reason != "" {
		t.Fatalf("reason = %q, want an empty reason", reason)
	}
}

func TestIsCurrentOnAMatchingRevisionAndACleanTree(t *testing.T) {
	root := newRepo(t)
	head := headRevision(t, root)

	clean := true
	current, reason := IsCurrent(root, Result{HeadRev: head, TreeClean: &clean})
	if !current {
		t.Fatalf("IsCurrent() = false, %q", reason)
	}
}

func TestIsCurrentOnARevisionMismatch(t *testing.T) {
	root := newRepo(t)

	clean := true
	current, reason := IsCurrent(root, Result{HeadRev: "0000000000000000000000000000000000000000", TreeClean: &clean})
	if current {
		t.Fatal("IsCurrent() = true, want false for a revision mismatch")
	}
	if reason == "" {
		t.Fatal("reason is empty, want a stated reason")
	}
}

func TestIsCurrentOnARunThatVerifiedADirtyTree(t *testing.T) {
	root := newRepo(t)
	head := headRevision(t, root)

	dirty := false
	current, _ := IsCurrent(root, Result{HeadRev: head, TreeClean: &dirty})
	if current {
		t.Fatal("IsCurrent() = true, want false; a dirty tree never yields pass")
	}
}

func TestIsCurrentIgnoresAWriteUnderTheStateDirectory(t *testing.T) {
	root := newRepo(t)
	head := headRevision(t, root)

	if err := os.MkdirAll(filepath.Join(root, StateDirectory, "changes", "SP-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, StateDirectory, "changes", "SP-1", "evidence-pack.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	clean := true
	current, reason := IsCurrent(root, Result{HeadRev: head, TreeClean: &clean})
	if !current {
		t.Fatalf("IsCurrent() = false, %q; a write under %s must not make a run stale", reason, StateDirectory)
	}
}

func TestIsCurrentSeesAWriteOutsideTheStateDirectory(t *testing.T) {
	root := newRepo(t)
	head := headRevision(t, root)

	if err := os.WriteFile(filepath.Join(root, "new.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	clean := true
	current, _ := IsCurrent(root, Result{HeadRev: head, TreeClean: &clean})
	if current {
		t.Fatal("IsCurrent() = true, want false; new untracked source is a real difference")
	}
}

func TestTreeStateReportsUnknownOutsideARepository(t *testing.T) {
	t.Parallel()

	head, clean := TreeState(t.TempDir())
	if head != "" || clean != nil {
		t.Fatalf("TreeState() = %q, %v; want an unknown state outside a repository", head, clean)
	}
}

// newRepo builds a Git repository with one commit and returns its root.
func newRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "init"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	return root
}

func headRevision(t *testing.T, root string) string {
	t.Helper()

	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(output[:len(output)-1])
}
