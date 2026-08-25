package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallClaudeCreatesCanonicalAndHarnessSkills(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	result, err := Install(root, TargetClaude, false, false)
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
	if _, err := Install(root, TargetCursor, false, false); err != nil {
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
	if _, err := Install(root, TargetClaude, false, false); err != nil {
		t.Fatalf("first Install: %v", err)
	}
	path := filepath.Join(root, ".claude", "skills", "shape-prd", "SKILL.md")
	if err := os.WriteFile(path, []byte("custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(root, TargetClaude, false, false); err == nil {
		t.Fatal("second Install expected overwrite protection error")
	}
}

func TestInstallRemovesRetiredSkill(t *testing.T) {
	root := t.TempDir()

	stale := filepath.Join(root, ".claude", "skills", "verify-change")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "SKILL.md"), []byte("# retired\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Install(root, TargetClaude, false, false)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("retired skill directory still exists: %v", err)
	}
	if len(result.Retired) == 0 {
		t.Fatal("Retired is empty; the result must report the removal")
	}
}

func TestInstallKeepsRetiredSkillOnRequest(t *testing.T) {
	root := t.TempDir()

	stale := filepath.Join(root, ".claude", "skills", "verify-change")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, "SKILL.md"), []byte("# retired\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(root, TargetClaude, false, true); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("retired skill directory was removed despite keepRetired: %v", err)
	}
}
