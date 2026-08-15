package skills

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newRun(condition string) EvalRun {
	return EvalRun{
		SchemaVersion:     "0.1",
		CaseID:            "prd-ready-stop",
		Skill:             "shape-prd",
		Condition:         condition,
		RecordedAt:        time.Now().UTC().Format(time.RFC3339),
		TaskSuccess:       true,
		QuestionsAsked:    2,
		HumanIntervention: false,
	}
}

func TestEvalRunValidate(t *testing.T) {
	t.Parallel()

	if err := newRun("without").Validate(); err != nil {
		t.Errorf("expected valid run, got %v", err)
	}
}

func TestEvalRunValidateBadCondition(t *testing.T) {
	t.Parallel()

	run := newRun("with-skill")
	if err := run.Validate(); err == nil {
		t.Error("expected error for invalid condition")
	}
}

func TestEvalRunValidateBadRate(t *testing.T) {
	t.Parallel()

	run := newRun("without")
	rate := 1.5
	run.BlockerRecall = &rate
	if err := run.Validate(); err == nil {
		t.Error("expected error for rate above 1")
	}
}

func TestEvalRunValidateBadTimestamp(t *testing.T) {
	t.Parallel()

	run := newRun("without")
	run.RecordedAt = "yesterday"
	if err := run.Validate(); err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

func TestRecordRunAndLoadRuns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	path, err := RecordRun(root, newRun("without"))
	if err != nil {
		t.Fatalf("RecordRun() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("run file missing: %v", err)
	}

	second, err := RecordRun(root, newRun("candidate"))
	if err != nil {
		t.Fatalf("second RecordRun() error = %v", err)
	}
	if second == path {
		t.Error("expected distinct paths for the second run")
	}

	runs, err := LoadRuns(root, "prd-ready-stop")
	if err != nil {
		t.Fatalf("LoadRuns() error = %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
}

func TestLoadRunsEmpty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runs, err := LoadRuns(root, "missing-case")
	if err != nil {
		t.Fatalf("LoadRuns() error = %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected no runs, got %d", len(runs))
	}
}

func TestRegressionDetectsTaskFailure(t *testing.T) {
	t.Parallel()

	baseline := newRun("without")
	candidate := newRun("candidate")
	candidate.TaskSuccess = false

	runs := []EvalRun{baseline, candidate}
	isRegression, reason := Regression(runs)
	if !isRegression {
		t.Error("expected regression when candidate fails a task baseline passed")
	}
	if reason == "" {
		t.Error("expected a regression reason")
	}
}

func TestRegressionDetectsRecallDrop(t *testing.T) {
	t.Parallel()

	baseline := newRun("previous")
	recallHigh := 1.0
	baseline.BlockerRecall = &recallHigh

	candidate := newRun("candidate")
	recallLow := 0.5
	candidate.BlockerRecall = &recallLow

	runs := []EvalRun{baseline, candidate}
	isRegression, reason := Regression(runs)
	if !isRegression {
		t.Error("expected regression when blocker recall decreases")
	}
	if reason == "" {
		t.Error("expected a regression reason")
	}
}

func TestRegressionNoChange(t *testing.T) {
	t.Parallel()

	baseline := newRun("without")
	candidate := newRun("candidate")

	runs := []EvalRun{baseline, candidate}
	isRegression, _ := Regression(runs)
	if isRegression {
		t.Error("expected no regression for equal results")
	}
}

func TestRegressionMissingCandidate(t *testing.T) {
	t.Parallel()

	runs := []EvalRun{newRun("without")}
	isRegression, _ := Regression(runs)
	if isRegression {
		t.Error("expected no regression when no candidate run exists")
	}
}

func TestRecordRunStoresFileInBenchmarks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path, err := RecordRun(root, newRun("candidate"))
	if err != nil {
		t.Fatalf("RecordRun() error = %v", err)
	}

	expected := filepath.Join(root, "benchmarks", "skill-evals", "prd-ready-stop", "candidate", "run-1.json")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}
