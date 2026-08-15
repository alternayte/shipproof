package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanCreate(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	source := filepath.Join(root, "delivery-plan.md")
	if err := os.WriteFile(source, []byte("# Delivery plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"plan", "create", source}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Created plan delivery-plan") {
		t.Errorf("expected created line, got: %s", stdout.String())
	}

	recordPath := filepath.Join(root, ".shipproof", "plans", "delivery-plan", "plan.json")
	if _, err := os.Stat(recordPath); err != nil {
		t.Errorf("expected plan record: %v", err)
	}
}

func TestPlanCreateMissingFile(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"plan", "create", filepath.Join(root, "missing.md")}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func TestPlanReviewEmpty(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"plan", "review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No plans found") {
		t.Errorf("expected empty message, got: %s", stdout.String())
	}
}

func TestPlanReviewValid(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	source := filepath.Join(root, "delivery-plan.md")
	if err := os.WriteFile(source, []byte("# Delivery plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	code := Run([]string{"plan", "create", source}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("plan create failed with %d", code)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code = Run([]string{"plan", "review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "delivery-plan: valid") {
		t.Errorf("expected valid line, got: %s", stdout.String())
	}
}

func TestPlanReviewReportsStaleSource(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	source := filepath.Join(root, "delivery-plan.md")
	if err := os.WriteFile(source, []byte("# Delivery plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	if code := Run([]string{"plan", "create", source}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("plan create failed with %d", code)
	}

	if err := os.WriteFile(source, []byte("# Delivery plan\n\nRevised.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := &bytes.Buffer{}
	code := Run([]string{"plan", "review"}, stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "source changed since the snapshot") {
		t.Errorf("expected stale message, got: %s", stdout.String())
	}
}

func TestPlanSyncRequiresLinear(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"plan", "sync", "--linear"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1 for missing issues file, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "issues.json") {
		t.Errorf("expected issues.json error, got: %s", stderr.String())
	}
}

func TestPlanUnknownSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"plan", "bogus"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}
