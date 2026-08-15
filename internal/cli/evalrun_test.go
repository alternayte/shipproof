package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEvalResultFile(t *testing.T, dir, name string, content map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSkillEvalRecord(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	resultFile := writeEvalResultFile(t, t.TempDir(), "result.json", map[string]any{
		"schema_version":     "0.1",
		"task_success":       true,
		"questions_asked":    2,
		"human_intervention": false,
	})

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"skill", "eval", "record", "prd-hidden-solution", "--condition", "without", "--file", resultFile}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	stored := filepath.Join(root, "benchmarks", "skill-evals", "prd-hidden-solution", "without", "run-1.json")
	if _, err := os.Stat(stored); err != nil {
		t.Errorf("expected recorded run: %v", err)
	}
	if !strings.Contains(stdout.String(), "Recorded eval run") {
		t.Errorf("expected record message, got: %s", stdout.String())
	}
}

func TestSkillEvalRecordUnknownCase(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	resultFile := writeEvalResultFile(t, t.TempDir(), "result.json", map[string]any{
		"schema_version":     "0.1",
		"task_success":       true,
		"questions_asked":    0,
		"human_intervention": false,
	})

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"skill", "eval", "record", "no-such-case", "--condition", "without", "--file", resultFile}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestSkillEvalRecordInvalidResult(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	resultFile := writeEvalResultFile(t, t.TempDir(), "result.json", map[string]any{
		"schema_version":     "0.1",
		"task_success":       true,
		"questions_asked":    -3,
		"human_intervention": false,
	})

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"skill", "eval", "record", "prd-hidden-solution", "--condition", "without", "--file", resultFile}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestSkillEvalResultsEmpty(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"skill", "eval", "results"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No recorded eval runs") {
		t.Errorf("expected empty message, got: %s", stdout.String())
	}
}

func TestSkillEvalRecordUsage(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"skill", "eval", "record", "case-only"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}
