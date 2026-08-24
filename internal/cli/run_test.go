package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
)

type fakeRunner struct {
	status agent.RunnerStatus
	result agent.RunResult
	runs   int
}

func (runner *fakeRunner) Probe(ctx context.Context) (agent.RunnerStatus, error) {
	return runner.status, nil
}

func (runner *fakeRunner) Run(ctx context.Context, req agent.RunRequest) (agent.RunResult, error) {
	runner.runs++
	return runner.result, nil
}

type fakeVerifier struct{ passed bool }

func (verifier fakeVerifier) Verify(ctx context.Context, changeID string) (agent.VerificationOutcome, error) {
	if verifier.passed {
		return agent.VerificationOutcome{Passed: true}, nil
	}
	return agent.VerificationOutcome{Passed: false, ExitCode: 1, Detail: "verification failed"}, nil
}

type fakeRevisions struct{ values []string }

func (revisions *fakeRevisions) Revision(ctx context.Context) (string, error) {
	value := revisions.values[0]
	if len(revisions.values) > 1 {
		revisions.values = revisions.values[1:]
	}
	return value, nil
}

func readyStatus() agent.RunnerStatus {
	return agent.RunnerStatus{
		Installed:     true,
		Authenticated: true,
		Version:       "1.0.0",
		Capabilities:  agent.RunnerCapabilities{ReadOnly: true, WorkspaceWrite: true},
	}
}

func withExecutor(t *testing.T, executor agent.Executor) {
	t.Helper()
	previous := executorFactory
	executorFactory = func(root, runnerOverride string) (agent.Executor, error) {
		return executor, nil
	}
	t.Cleanup(func() { executorFactory = previous })
}

func setupRunChange(t *testing.T) (string, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestChangeRecord(t, root, "CH-100")
	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
	return root, &bytes.Buffer{}, &bytes.Buffer{}
}

// P8: `shipproof run <change-id>` returns PASS.
func TestRunReturnsPass(t *testing.T) {
	root, stdout, stderr := setupRunChange(t)
	reviewer := &fakeRunner{
		status: readyStatus(),
		result: agent.RunResult{Status: agent.RunStatusSuccess, Summary: "- The retry path can duplicate a delivery."},
	}
	withExecutor(t, agent.Executor{
		Runner:           &fakeRunner{status: readyStatus(), result: agent.RunResult{Status: agent.RunStatusSuccess}},
		RunnerName:       "stub",
		ReviewRunner:     reviewer,
		ReviewRunnerName: "stub-reviewer",
		Verifier:         fakeVerifier{passed: true},
		Revisions:        &fakeRevisions{values: []string{"base1", "head1"}},
		MaxAttempts:      2,
	})

	if code := Run([]string{"run", "CH-100"}, stdout, stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "PASS") {
		t.Fatalf("output = %s", output)
	}
	if !strings.Contains(output, "inferred") {
		t.Fatalf("reviewer findings must be reported as inferred: %s", output)
	}

	data, err := os.ReadFile(filepath.Join(root, ".shipproof", "changes", "CH-100", "execution.json"))
	if err != nil {
		t.Fatalf("read execution record: %v", err)
	}
	var record agent.ExecutionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("parse execution record: %v", err)
	}
	if record.Status != agent.OutcomePass {
		t.Fatalf("record status = %q", record.Status)
	}
	if record.Execution.BaseRevision != "base1" || record.Execution.ResultRevision != "head1" {
		t.Fatalf("record execution = %+v", record.Execution)
	}
}

// P8: `shipproof run <change-id>` returns NEEDS_REVIEW.
func TestRunReturnsNeedsReview(t *testing.T) {
	_, stdout, stderr := setupRunChange(t)
	withExecutor(t, agent.Executor{
		Runner:      &fakeRunner{status: readyStatus(), result: agent.RunResult{Status: agent.RunStatusSuccess, Summary: "All tests pass."}},
		RunnerName:  "stub",
		Verifier:    fakeVerifier{passed: false},
		Revisions:   &fakeRevisions{values: []string{"base1"}},
		MaxAttempts: 2,
	})

	if code := Run([]string{"run", "CH-100"}, stdout, stderr); code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "NEEDS_REVIEW") {
		t.Fatalf("output = %s", stdout.String())
	}
}

// P8: `shipproof run <change-id>` returns BLOCKED.
func TestRunReturnsBlocked(t *testing.T) {
	_, stdout, stderr := setupRunChange(t)
	withExecutor(t, agent.Executor{
		Runner:      &fakeRunner{status: agent.RunnerStatus{Installed: true, Detail: "Run `codex login`."}},
		RunnerName:  "stub",
		Verifier:    fakeVerifier{passed: true},
		Revisions:   &fakeRevisions{values: []string{"base1"}},
		MaxAttempts: 1,
	})

	if code := Run([]string{"run", "CH-100"}, stdout, stderr); code != 2 {
		t.Fatalf("exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "BLOCKED") {
		t.Fatalf("output = %s", stdout.String())
	}
}

func TestRunRejectsMissingChange(t *testing.T) {
	_, stdout, stderr := setupRunChange(t)
	withExecutor(t, agent.Executor{})
	if code := Run([]string{"run", "CH-404"}, stdout, stderr); code != 1 {
		t.Fatalf("exit = %d", code)
	}
}

func TestRunRequiresChangeID(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := Run([]string{"run"}, stdout, stderr); code != 2 {
		t.Fatalf("exit = %d", code)
	}
}
