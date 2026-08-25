package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNextReportsNoChange(t *testing.T) {
	newCLITestRoot(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "SP-999"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "NO_CHANGE") {
		t.Fatalf("stdout does not name the phase:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "prepare-change") {
		t.Fatalf("stdout does not name the skill:\n%s", stdout.String())
	}
}

func TestNextJSONOutput(t *testing.T) {
	newCLITestRoot(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "SP-999", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	var payload struct {
		ChangeID    string `json:"change_id"`
		Phase       string `json:"phase"`
		NextCommand string `json:"next_command"`
		NextSkill   string `json:"next_skill"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Phase != "NO_CHANGE" {
		t.Fatalf("phase = %q, want %q", payload.Phase, "NO_CHANGE")
	}
	if payload.ChangeID != "SP-999" {
		t.Fatalf("change_id = %q, want %q", payload.ChangeID, "SP-999")
	}
}

func TestNextRejectsUnrecognizedOption(t *testing.T) {
	newCLITestRoot(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"next", "--Json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown option") {
		t.Fatalf("stderr does not name the problem:\n%s", stderr.String())
	}
}

func TestNextWithSeveralOpenChangesListsThem(t *testing.T) {
	root := newCLITestRoot(t)

	for _, id := range []string{"SP-400", "SP-401"} {
		source := filepath.Join(root, id+".md")
		if err := os.WriteFile(source, []byte("# "+id+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		if code := Run([]string{"change", "start", id, "--source", source, "--ceremony", "0"}, &out, &errOut); code != 0 {
			t.Fatalf("change start %s failed: %s", id, errOut.String())
		}
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"next"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected a non-zero exit code for several open changes")
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "SP-400") || !strings.Contains(combined, "SP-401") {
		t.Fatalf("output does not list both changes:\n%s", combined)
	}
}

func TestNextWithNoArgumentResolvesTheSoleOpenChange(t *testing.T) {
	root := newCLITestRoot(t)

	source := filepath.Join(root, "SP-402.md")
	if err := os.WriteFile(source, []byte("# SP-402\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := Run([]string{"change", "start", "SP-402", "--source", source, "--ceremony", "0"}, &out, &errOut); code != 0 {
		t.Fatalf("change start failed: %s", errOut.String())
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"next"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "NEEDS_RUN") {
		t.Fatalf("stdout does not name the phase:\n%s", stdout.String())
	}
}

// newCLITestRoot creates a repository root and points the CLI at it through
// the RunOverrides hook that internal/cli/doc.go:105 already provides. The
// existing tests in change_test.go and evidence_test.go use the same hook.
// RunOverrides is shared process state, so these tests must not call
// t.Parallel().
func newCLITestRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".shipproof", "config.yaml"), []byte("version: 1\nverification:\n  command: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
	return root
}

func TestNextIgnoresADirectoryWithoutAChangeRecord(t *testing.T) {
	root := newCLITestRoot(t)

	source := filepath.Join(root, "SP-410.md")
	if err := os.WriteFile(source, []byte("# SP-410\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := Run([]string{"change", "start", "SP-410", "--source", source, "--ceremony", "0"}, &out, &errOut); code != 0 {
		t.Fatalf("change start failed: %s", errOut.String())
	}

	// shipproof verification init creates such a directory before any change
	// record exists. It must not count as an open change.
	phantom := filepath.Join(root, ".shipproof", "changes", "SP-411")
	if err := os.MkdirAll(phantom, 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"next"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SP-410") {
		t.Fatalf("stdout does not resolve the sole open change:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), "SP-411") {
		t.Fatalf("the phantom directory counted as an open change:\n%s%s", stdout.String(), stderr.String())
	}
}

func TestNextSurfacesACorruptChangeRecord(t *testing.T) {
	root := newCLITestRoot(t)

	source := filepath.Join(root, "SP-420.md")
	if err := os.WriteFile(source, []byte("# SP-420\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := Run([]string{"change", "start", "SP-420", "--source", source, "--ceremony", "0"}, &out, &errOut); code != 0 {
		t.Fatalf("change start failed: %s", errOut.String())
	}

	corrupt := filepath.Join(root, ".shipproof", "changes", "SP-421")
	if err := os.MkdirAll(corrupt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corrupt, "change.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"next"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected a non-zero exit code for a corrupt change record:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "SP-421") {
		t.Fatalf("stderr does not name the corrupt change:\n%s", stderr.String())
	}
}
