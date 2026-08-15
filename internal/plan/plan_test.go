package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndLoad(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "delivery-plan.md")
	content := "# Delivery plan\n\nThree changes.\n"
	if err := os.WriteFile(source, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Create(root, source)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if record.PlanID != "delivery-plan" {
		t.Errorf("plan id = %q, want delivery-plan", record.PlanID)
	}

	loaded, err := Load(root, record.PlanID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.SHA256 != record.SHA256 {
		t.Errorf("sha256 mismatch after load")
	}

	if err := loaded.VerifyHash(root); err != nil {
		t.Errorf("VerifyHash() error = %v", err)
	}

	stale, current, err := loaded.Staleness(root)
	if err != nil {
		t.Fatalf("Staleness() error = %v", err)
	}
	if stale {
		t.Errorf("expected current plan, got stale with hash %s", current)
	}
}

func TestCreateDuplicateFails(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "delivery-plan.md")
	if err := os.WriteFile(source, []byte("# Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Create(root, source); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if _, err := Create(root, source); err == nil {
		t.Fatal("expected duplicate create to fail")
	}
}

func TestStalenessAfterSourceChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "delivery-plan.md")
	if err := os.WriteFile(source, []byte("# Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Create(root, source)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := os.WriteFile(source, []byte("# Plan\n\nRevised.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, current, err := record.Staleness(root)
	if err != nil {
		t.Fatalf("Staleness() error = %v", err)
	}
	if !stale {
		t.Error("expected stale plan after source change")
	}
	if current == record.SHA256 {
		t.Error("current hash must differ from the recorded snapshot hash")
	}
}

func TestStalenessAfterSourceRemoval(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "delivery-plan.md")
	if err := os.WriteFile(source, []byte("# Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	record, err := Create(root, source)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}

	stale, current, err := record.Staleness(root)
	if err != nil {
		t.Fatalf("Staleness() error = %v", err)
	}
	if !stale {
		t.Error("expected stale plan when the source is missing")
	}
	if current != "" {
		t.Errorf("expected empty current hash, got %q", current)
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"one.md", "two.md"} {
		source := filepath.Join(root, name)
		if err := os.WriteFile(source, []byte("# Plan\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Create(root, source); err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
	}

	ids, err := List(root)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(ids))
	}
}

func TestInvalidPlanIDSource(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "Delivery Plan!.md")
	if err := os.WriteFile(source, []byte("# Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Create(root, source); err == nil {
		t.Fatal("expected error for a source name that cannot become a plan id")
	}
}
