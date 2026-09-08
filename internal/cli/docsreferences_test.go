package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cutReferences names every word that SP-023 removed from the product. The
// user-facing documentation must not name any of them.
var cutReferences = []string{
	"shipproof doc ",
	"shipproof shape ",
	"shipproof plan ",
	"shipproof linear ",
	"shipproof skill eval",
	"shipproof evidence review",
	"shape-prd",
	"shape-sdd",
	"review-prd",
	"review-sdd",
	"decompose-plan",
	"triage-change",
	"record-decision",
	"benchmark-run",
	"skill-evals",

	// SP-024 folded these verbs into the five commands of Section 4.
	"shipproof verification",
	"shipproof verify",
	"shipproof harness",
	"shipproof change ",
	"shipproof next",
	"shipproof coverage",
	"shipproof skill",
	"shipproof evidence",
	"shipproof review ",
	"shipproof telemetry",
	"shipproof report",

	// SP-033 replaced the six skill packages with the three instruction files
	// of Section 8.2.
	"prepare-change",
	"plan-verification",
	"implement-change",
	"produce-evidence",
	"review-change",
	"prepare-human-review",
}

// documentedFiles names the user-facing documents. CHANGELOG.md is a
// historical record, so it keeps its old words.
var documentedFiles = []string{
	"README.md",
	"docs/workflow.md",
	"docs/adoption.md",
	"benchmarks/README.md",
	"skills/README.md",
	"AGENTS.md",
	"docs/hooks.md",
	"docs/controls.md",
}

func TestDocumentationNamesNoCutFeature(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range documentedFiles {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		for _, word := range cutReferences {
			if strings.Contains(text, word) {
				t.Errorf("%s still names %q", name, word)
			}
		}
	}
}
