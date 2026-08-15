package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangeStart(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	src := filepath.Join(root, "change-src.md")
	if err := os.WriteFile(src, []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"change", "start", "SP-030", "--source", src, "--shaping", "my-session"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Shaping: my-session") {
		t.Errorf("expected shaping line in output, got: %s", output)
	}

	recordPath := filepath.Join(root, ".shipproof", "changes", "SP-030", "change.json")
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("read change record: %v", err)
	}
	var record struct {
		ShapingRef string `json:"shaping_ref"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("parse change record: %v", err)
	}
	if record.ShapingRef != "my-session" {
		t.Errorf("shaping_ref = %q, want my-session", record.ShapingRef)
	}
}

func TestChangeStartWithoutShaping(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	src := filepath.Join(root, "change-src.md")
	if err := os.WriteFile(src, []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"change", "start", "SP-031", "--source", src}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	recordPath := filepath.Join(root, ".shipproof", "changes", "SP-031", "change.json")
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("read change record: %v", err)
	}
	var record struct {
		ShapingRef string `json:"shaping_ref"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("parse change record: %v", err)
	}
	if record.ShapingRef != "" {
		t.Errorf("shaping_ref should be empty, got %q", record.ShapingRef)
	}
}

func TestChangeStartShapingFlagMissingValue(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"change", "start", "SP-032", "--source", "x.md", "--shaping"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}
