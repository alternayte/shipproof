package cli

import (
	"bytes"
	"strings"
	"testing"
)

// removedVerbs lists every command that SP-023 cut. Each one must exit with
// code 2 and print exactly one line.
var removedVerbs = [][]string{
	{"doc"},
	{"doc", "review", "some.md"},
	{"shape"},
	{"shape", "prd", "subject"},
	{"plan"},
	{"plan", "create", "some.md"},
	{"linear"},
	{"linear", "issue", "ABC-1"},
	{"evidence", "review"},
	{"evidence", "review", "SP-001"},
	{"skill", "eval"},
	{"skill", "eval", "list"},
}

func TestRemovedVerbsExitTwoWithOneLine(t *testing.T) {
	for _, args := range removedVerbs {
		name := strings.Join(args, " ")
		t.Run(name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			code := Run(args, stdout, stderr)
			if code != 2 {
				t.Fatalf("expected exit 2, got %d", code)
			}
			if stdout.Len() != 0 {
				t.Errorf("expected empty stdout, got: %q", stdout.String())
			}

			message := stderr.String()
			if !strings.HasSuffix(message, "\n") {
				t.Fatalf("expected a terminated line, got: %q", message)
			}
			lines := strings.Split(strings.TrimSuffix(message, "\n"), "\n")
			if len(lines) != 1 {
				t.Fatalf("expected one line, got %d: %q", len(lines), message)
			}
			if strings.TrimSpace(lines[0]) == "" {
				t.Fatal("expected a non-empty line")
			}
		})
	}
}

// cutWords lists the words that the usage text must no longer name.
var cutWords = []string{
	"shipproof doc",
	"shipproof shape",
	"shipproof plan",
	"shipproof linear",
	"shipproof skill eval",
	"evidence review",
}

func TestUsageNamesNoCutVerb(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if code := Run([]string{"help"}, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	usage := stdout.String()
	for _, word := range cutWords {
		if strings.Contains(usage, word) {
			t.Errorf("usage still names %q", word)
		}
	}
}
