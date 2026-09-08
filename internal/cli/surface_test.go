package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// publicVerbs holds the whole public surface of Section 4 of the SDD.
var publicVerbs = []string{
	"shipproof init",
	"shipproof start",
	"shipproof prove",
	"shipproof pack",
	"shipproof status",
	"shipproof runner",
	"shipproof config",
	"shipproof run ",
}

// foldedVerbs holds every verb that SP-024 folded into the public surface.
var foldedVerbs = []string{
	"verification", "verify", "harness", "change", "next",
	"coverage", "skill", "evidence", "review", "telemetry", "report",
}

func TestUsageNamesTheWholePublicSurface(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if code := Run([]string{"help"}, stdout, stderr); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	usage := stdout.String()

	for _, verb := range publicVerbs {
		if !strings.Contains(usage, verb) {
			t.Errorf("usage does not name %q", verb)
		}
	}
	for _, verb := range foldedVerbs {
		if strings.Contains(usage, "shipproof "+verb) {
			t.Errorf("usage still names the folded verb %q", verb)
		}
	}
}

func TestFoldedVerbsExitTwoWithOneLine(t *testing.T) {
	for _, verb := range foldedVerbs {
		t.Run(verb, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			code := Run([]string{verb}, stdout, stderr)
			if code != 2 {
				t.Fatalf("expected exit 2, got %d", code)
			}
			if stdout.Len() != 0 {
				t.Errorf("expected empty stdout, got: %q", stdout.String())
			}
			message := stderr.String()
			lines := strings.Split(strings.TrimSuffix(message, "\n"), "\n")
			if len(lines) != 1 || strings.TrimSpace(lines[0]) == "" {
				t.Fatalf("expected one non-empty line, got: %q", message)
			}
		})
	}
}

// TestStartThenPackReachesAFullResult covers SP-024-R8 and Section 14.2 C5.
func TestStartThenPackReachesAFullResult(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	intent := filepath.Join(root, "SP-500.md")
	body := "# SP-500 — Example\n\n### SP-500-R1 — The tool records the intent\n\nThe tool records the intent.\n"
	if err := os.WriteFile(intent, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"start", "SP-500", "--intent", intent}, stdout, stderr); code != 0 {
		t.Fatalf("start: expected exit 0, got %d: %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"pack", "SP-500"}, stdout, stderr); code != 0 {
		t.Fatalf("pack: expected exit 0, got %d: %s", code, stderr.String())
	}

	packPath := filepath.Join(root, ".shipproof", "changes", "SP-500", "evidence-pack.json")
	if _, err := os.Stat(packPath); err != nil {
		t.Fatalf("expected an evidence pack: %v", err)
	}
}
