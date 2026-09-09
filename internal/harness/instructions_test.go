package harness

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// neverWriteRule is the rule that Section 8.2 calls the one that matters most.
const neverWriteRule = "An agent must never write a result that a tool did not produce."

// everyTarget lists the five harness targets of Section 8.1.
var everyTarget = []Target{TargetClaude, TargetCursor, TargetCodex, TargetOpenCode, TargetAgents}

// TestInstallWritesThreeInstructionFiles is the proof for row T1.
func TestInstallWritesThreeInstructionFiles(t *testing.T) {
	for _, target := range everyTarget {
		root := t.TempDir()
		if _, err := Install(root, target, false, false); err != nil {
			t.Fatalf("Install %s: %v", target, err)
		}
		for _, base := range []string{
			filepath.Join(root, ".shipproof", "skills"),
			targetDirectory(root, target),
		} {
			names := listFiles(t, base)
			want := []string{"capture-intent", "plan-proof", "read-evidence"}
			if len(names) != len(want) {
				t.Fatalf("%s holds %v, want %v", base, names, want)
			}
			for index, name := range names {
				if name != want[index] {
					t.Fatalf("%s holds %v, want %v", base, names, want)
				}
			}
		}
	}
}

// TestEveryInstructionFileHoldsTheRule holds requirement R2.
func TestEveryInstructionFileHoldsTheRule(t *testing.T) {
	root := t.TempDir()
	if _, err := Install(root, TargetClaude, false, false); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, ".shipproof", "skills")
	for _, name := range listFiles(t, base) {
		if !strings.Contains(string(readInstruction(t, base, name)), neverWriteRule) {
			t.Errorf("%s does not hold the rule that matters most", name)
		}
	}
}

// TestInstallRemovesARetiredSkill holds requirement R4. A retired skill left
// in a harness directory still answers to an agent.
func TestInstallRemovesARetiredSkill(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"prepare-change", "produce-evidence", "review-change"} {
		directory := filepath.Join(root, ".claude", "skills", name)
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := Install(root, TargetClaude, false, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(result.Retired) == 0 {
		t.Fatal("Install retired no skill directory")
	}
	for _, name := range []string{"prepare-change", "produce-evidence", "review-change"} {
		if _, err := os.Stat(filepath.Join(root, ".claude", "skills", name)); !os.IsNotExist(err) {
			t.Errorf("the retired skill %s survives", name)
		}
	}
}

// commandPattern reads every `shipproof ...` command from an instruction file.
var commandPattern = regexp.MustCompile("`shipproof ([a-z-]+)")

// TestInstructionsNameOnlyTheSurface holds requirement R5. Section 4 fixes the
// surface, and an instruction file must never send an agent to a removed verb.
func TestInstructionsNameOnlyTheSurface(t *testing.T) {
	surface := map[string]bool{
		"init": true, "start": true, "prove": true, "pack": true, "status": true,
		"runner": true, "config": true, "run": true, "version": true,
	}

	root := t.TempDir()
	if _, err := Install(root, TargetClaude, false, false); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, ".shipproof", "skills")
	for _, name := range listFiles(t, base) {
		for _, match := range commandPattern.FindAllStringSubmatch(string(readInstruction(t, base, name)), -1) {
			if !surface[match[1]] {
				t.Errorf("%s names the command %q, which Section 4 does not hold", name, match[1])
			}
		}
	}
}

// listFiles names each installed instruction. SP-041 moved every instruction
// into its own package, because a loose Markdown file in a skills directory is
// not loaded by any harness. Section 8.2 still names three instructions.
func listFiles(t *testing.T, base string) []string {
	t.Helper()
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read %s: %v", base, err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			t.Fatalf("%s holds the loose file %s, which no harness reads", base, entry.Name())
		}
		names = append(names, entry.Name())
	}
	return names
}

// readInstruction returns the body of one installed instruction.
func readInstruction(t *testing.T, base, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, name, "SKILL.md"))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return data
}
