package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/verification"
)

// TestEveryInstructionIsASkillPackage holds requirement R1 of SP-041. A loose
// Markdown file in a skills directory is not loaded by the harness, so the
// agent never sees it. Delivery is not arrival.
func TestEveryInstructionIsASkillPackage(t *testing.T) {
	for _, target := range everyTarget {
		root := t.TempDir()
		if _, err := Install(root, target, false, false); err != nil {
			t.Fatalf("Install %s: %v", target, err)
		}
		base := targetDirectory(root, target)
		entries, err := os.ReadDir(base)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 3 {
			t.Fatalf("%s holds %d entries, want 3", base, len(entries))
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				t.Errorf("%s holds the loose file %s, which no harness loads", base, entry.Name())
				continue
			}
			skill := filepath.Join(base, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skill); err != nil {
				t.Errorf("%s holds no SKILL.md", entry.Name())
			}
		}
	}
}

// TestEverySkillCarriesFrontmatter holds requirement R2. Without a name and a
// description a harness cannot decide when the file applies.
func TestEverySkillCarriesFrontmatter(t *testing.T) {
	root := t.TempDir()
	if _, err := Install(root, TargetClaude, false, false); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, ".shipproof", "skills")
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(base, entry.Name(), "SKILL.md"))
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		body := string(data)
		if !strings.HasPrefix(body, "---\n") {
			t.Errorf("%s carries no frontmatter", entry.Name())
			continue
		}
		front := body[4 : strings.Index(body[4:], "---")+4]
		for _, field := range []string{"name:", "description:"} {
			if !strings.Contains(front, field) {
				t.Errorf("%s frontmatter holds no %s", entry.Name(), field)
			}
		}
		if !strings.Contains(front, "name: "+entry.Name()) {
			t.Errorf("%s frontmatter names something else", entry.Name())
		}
	}
}

// TestPlanProofNamesTheFileToWrite holds requirements R3 and R4. An agent that
// learns the philosophy and cannot find the file has learned nothing usable.
func TestPlanProofNamesTheFileToWrite(t *testing.T) {
	root := t.TempDir()
	if _, err := Install(root, TargetClaude, false, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "skills", "plan-proof", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{
		"verification.json",
		`"schema_version"`,
		`"requirements"`,
		`"proof"`,
		`"command"`,
		`"human": true`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("plan-proof does not show %s", want)
		}
	}
}

// TestInstallRemovesALooseInstructionFile holds requirement R6. A file that no
// harness reads is worse than no file, because it looks installed.
func TestInstallRemovesALooseInstructionFile(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, ".claude", "skills")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"capture-intent.md", "plan-proof.md", "read-evidence.md"} {
		if err := os.WriteFile(filepath.Join(base, name), []byte("# old\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := Install(root, TargetClaude, false, false); err != nil {
		t.Fatalf("Install: %v", err)
	}
	for _, name := range []string{"capture-intent.md", "plan-proof.md", "read-evidence.md"} {
		if _, err := os.Stat(filepath.Join(base, name)); !os.IsNotExist(err) {
			t.Errorf("the loose file %s survives, and no harness reads it", name)
		}
	}
}

// TestTheWorkedExampleIsAValidPlan holds the rule that an example a reader
// copies must work. A worked example that does not parse teaches a shape the
// tool rejects.
func TestTheWorkedExampleIsAValidPlan(t *testing.T) {
	root := t.TempDir()
	if _, err := Install(root, TargetClaude, false, false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "skills", "plan-proof", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Take the first fenced json block.
	start := strings.Index(string(data), "```json\n")
	if start < 0 {
		t.Fatal("plan-proof holds no json example")
	}
	rest := string(data)[start+len("```json\n"):]
	end := strings.Index(rest, "```")
	if end < 0 {
		t.Fatal("the json example is not closed")
	}
	example := rest[:end]

	var plan verification.Plan
	if err := json.Unmarshal([]byte(example), &plan); err != nil {
		t.Fatalf("the worked example is not valid JSON: %v", err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("the worked example does not answer to the plan schema: %v", err)
	}
	if len(plan.Requirements) < 2 {
		t.Fatal("the example shows fewer than two requirements, so it cannot show both proof kinds")
	}
	var sawCommand, sawHuman bool
	for _, item := range plan.Requirements {
		for _, proof := range item.Proof {
			if proof.IsHuman() {
				sawHuman = true
			}
			if proof.IsAutomated() {
				sawCommand = true
			}
		}
	}
	if !sawCommand || !sawHuman {
		t.Errorf("the example shows command=%v human=%v; it must show both", sawCommand, sawHuman)
	}
}
