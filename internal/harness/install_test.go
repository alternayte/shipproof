package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallClaudeCreatesCanonicalAndHarnessSkills(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	result, err := Install(root, TargetClaude, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if result.CanonicalCreated == 0 || result.HarnessCreated == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	for _, path := range []string{
		filepath.Join(root, ".shipproof", "skills", "shape-prd", "SKILL.md"),
		filepath.Join(root, ".claude", "skills", "shape-prd", "SKILL.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
}

func TestInstallCursorUsesPortableAgentsDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Install(root, TargetCursor, false); err != nil {
		t.Fatalf("Install: %v", err)
	}
	path := filepath.Join(root, ".agents", "skills", "review-sdd", "SKILL.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s: %v", path, err)
	}
}

func TestInstallDoesNotOverwriteModifiedSkill(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Install(root, TargetClaude, false); err != nil {
		t.Fatalf("first Install: %v", err)
	}
	path := filepath.Join(root, ".claude", "skills", "shape-prd", "SKILL.md")
	if err := os.WriteFile(path, []byte("custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(root, TargetClaude, false); err == nil {
		t.Fatal("second Install expected overwrite protection error")
	}
}
