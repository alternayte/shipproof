package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// scriptedRunner reports whatever a test tells it to report. It never touches
// the repository. It exists to prove that a runner claim is not evidence.
type scriptedRunner struct {
	status     RunnerStatus
	result     RunResult
	probeErr   error
	runErr     error
	runCount   int
	lastRole   AgentRole
	lastPrompt string
}

func (runner *scriptedRunner) Probe(ctx context.Context) (RunnerStatus, error) {
	return runner.status, runner.probeErr
}

func (runner *scriptedRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	runner.runCount++
	runner.lastRole = req.Role
	runner.lastPrompt = req.Instructions
	return runner.result, runner.runErr
}

type scriptedVerifier struct {
	outcomes []VerificationOutcome
	calls    int
}

func (verifier *scriptedVerifier) Verify(ctx context.Context, changeID string) (VerificationOutcome, error) {
	index := verifier.calls
	if index >= len(verifier.outcomes) {
		index = len(verifier.outcomes) - 1
	}
	verifier.calls++
	return verifier.outcomes[index], nil
}

type fixedRevisions struct {
	values []string
	calls  int
}

func (revisions *fixedRevisions) Revision(ctx context.Context) (string, error) {
	index := revisions.calls
	if index >= len(revisions.values) {
		index = len(revisions.values) - 1
	}
	revisions.calls++
	return revisions.values[index], nil
}

func usableStatus() RunnerStatus {
	return RunnerStatus{
		Installed:     true,
		Authenticated: true,
		Version:       "1.0.0",
		Capabilities:  RunnerCapabilities{ReadOnly: true, WorkspaceWrite: true},
	}
}

func newRequest() RunRequest {
	return RunRequest{
		Workspace:    "/workspace",
		Change:       Change{ID: "CH-100"},
		Instructions: "Implement the approved change.",
	}
}

func fixedNow() string { return "2026-08-24T10:00:00Z" }

// P9: a runner claim is never evidence.
func TestRunnerClaimIsNotEvidence(t *testing.T) {
	runner := &scriptedRunner{status: usableStatus(), result: RunResult{Status: RunStatusSuccess, Summary: "All tests pass."}}
	executor := Executor{
		Runner:      runner,
		RunnerName:  "stub",
		Verifier:    &scriptedVerifier{outcomes: []VerificationOutcome{{Passed: false, ExitCode: 1, Detail: "3 tests failed"}}},
		Revisions:   &fixedRevisions{values: []string{"base1", "head1"}},
		MaxAttempts: 1,
		Now:         fixedNow,
	}

	record, err := executor.Execute(context.Background(), newRequest())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if record.Status == OutcomePass {
		t.Fatal("a runner success claim must not produce PASS when verification fails")
	}
	if record.Status != OutcomeNeedsReview {
		t.Fatalf("status = %q, want NEEDS_REVIEW", record.Status)
	}
	if record.Attempts[0].RunnerStatus != string(RunStatusSuccess) {
		t.Fatalf("the runner claim must still be recorded: %+v", record.Attempts[0])
	}
	if record.Attempts[0].Verification != "fail" {
		t.Fatalf("verification must decide: %+v", record.Attempts[0])
	}
}

// P13: the repair loop is bounded.
func TestRepairLoopIsBounded(t *testing.T) {
	runner := &scriptedRunner{status: usableStatus(), result: RunResult{Status: RunStatusSuccess}}
	executor := Executor{
		Runner:      runner,
		RunnerName:  "stub",
		Verifier:    &scriptedVerifier{outcomes: []VerificationOutcome{{Passed: false, ExitCode: 1, Detail: "still failing"}}},
		Revisions:   &fixedRevisions{values: []string{"base1"}},
		MaxAttempts: 2,
		Now:         fixedNow,
	}

	record, err := executor.Execute(context.Background(), newRequest())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if runner.runCount != 2 {
		t.Fatalf("runner ran %d times, want 2", runner.runCount)
	}
	if len(record.Attempts) != 2 {
		t.Fatalf("attempts = %d, want 2", len(record.Attempts))
	}
	if record.Status != OutcomeNeedsReview {
		t.Fatalf("status = %q, want NEEDS_REVIEW", record.Status)
	}
	if !strings.Contains(runner.lastPrompt, "Do not widen the scope") {
		t.Fatalf("repair instructions must not widen scope: %q", runner.lastPrompt)
	}
}

// P11: an unenforceable role policy returns BLOCKED.
func TestUnenforceableReviewerPolicyBlocks(t *testing.T) {
	implementer := &scriptedRunner{status: usableStatus(), result: RunResult{Status: RunStatusSuccess}}
	reviewerStatus := usableStatus()
	reviewerStatus.Capabilities = RunnerCapabilities{WorkspaceWrite: true}
	reviewer := &scriptedRunner{status: reviewerStatus, result: RunResult{Status: RunStatusSuccess}}

	executor := Executor{
		Runner:           implementer,
		RunnerName:       "stub",
		ReviewRunner:     reviewer,
		ReviewRunnerName: "loose",
		Verifier:         &scriptedVerifier{outcomes: []VerificationOutcome{{Passed: true}}},
		Revisions:        &fixedRevisions{values: []string{"base1", "head1"}},
		MaxAttempts:      1,
		Now:              fixedNow,
	}

	record, err := executor.Execute(context.Background(), newRequest())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if record.Status != OutcomeBlocked {
		t.Fatalf("status = %q, want BLOCKED", record.Status)
	}
	if !strings.Contains(record.Detail, "read-only") {
		t.Fatalf("detail = %q", record.Detail)
	}
	if reviewer.runCount != 0 {
		t.Fatal("a reviewer that cannot enforce read-only must not run")
	}
}

func TestUnenforceableImplementerPolicyBlocks(t *testing.T) {
	status := usableStatus()
	status.Capabilities = RunnerCapabilities{ReadOnly: true}
	runner := &scriptedRunner{status: status}
	executor := Executor{
		Runner:      runner,
		RunnerName:  "stub",
		Verifier:    &scriptedVerifier{outcomes: []VerificationOutcome{{Passed: true}}},
		Revisions:   &fixedRevisions{values: []string{"base1"}},
		MaxAttempts: 1,
		Now:         fixedNow,
	}
	record, err := executor.Execute(context.Background(), newRequest())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if record.Status != OutcomeBlocked {
		t.Fatalf("status = %q, want BLOCKED", record.Status)
	}
	if runner.runCount != 0 {
		t.Fatal("the runner must not run under an unenforceable policy")
	}
}

func TestUnusableRunnerBlocks(t *testing.T) {
	runner := &scriptedRunner{status: RunnerStatus{Installed: true, Detail: "Run `codex login`."}}
	executor := Executor{
		Runner:      runner,
		RunnerName:  "stub",
		Verifier:    &scriptedVerifier{outcomes: []VerificationOutcome{{Passed: true}}},
		Revisions:   &fixedRevisions{values: []string{"base1"}},
		MaxAttempts: 1,
		Now:         fixedNow,
	}
	record, err := executor.Execute(context.Background(), newRequest())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if record.Status != OutcomeBlocked || !strings.Contains(record.Detail, "codex login") {
		t.Fatalf("record = %+v", record)
	}
}

// P12: adversarial reviewer findings are agent-inferred.
func TestReviewerFindingsAreInferred(t *testing.T) {
	implementer := &scriptedRunner{status: usableStatus(), result: RunResult{Status: RunStatusSuccess}}
	reviewer := &scriptedRunner{
		status: usableStatus(),
		result: RunResult{Status: RunStatusSuccess, Summary: "Findings:\n- The retry path can duplicate a delivery.\n- The rotation invariant is undocumented."},
	}
	executor := Executor{
		Runner:           implementer,
		RunnerName:       "stub",
		ReviewRunner:     reviewer,
		ReviewRunnerName: "reviewer-stub",
		Verifier:         &scriptedVerifier{outcomes: []VerificationOutcome{{Passed: true}}},
		Revisions:        &fixedRevisions{values: []string{"base1", "head1"}},
		MaxAttempts:      2,
		Now:              fixedNow,
	}

	record, err := executor.Execute(context.Background(), newRequest())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if record.Status != OutcomePass {
		t.Fatalf("status = %q, want PASS", record.Status)
	}
	if len(record.Findings) != 2 {
		t.Fatalf("findings = %+v", record.Findings)
	}
	for _, finding := range record.Findings {
		if finding.Provenance != ProvenanceInferred {
			t.Fatalf("finding provenance = %q, want inferred", finding.Provenance)
		}
		if finding.Source != "reviewer-stub" {
			t.Fatalf("finding source = %q", finding.Source)
		}
	}
	if reviewer.lastRole != RoleReviewer {
		t.Fatalf("reviewer ran as %q", reviewer.lastRole)
	}
}

// P10: the execution record is runner-neutral and durable.
func TestExecutionRecordIsRunnerNeutral(t *testing.T) {
	metaType := reflect.TypeOf(ExecutionMeta{})
	allowed := map[string]bool{
		"runner": true, "runner_version": true, "session_ref": true,
		"started_at": true, "completed_at": true, "base_revision": true,
		"result_revision": true, "attempt": true, "metadata": true,
	}
	for index := 0; index < metaType.NumField(); index++ {
		tag := strings.Split(metaType.Field(index).Tag.Get("json"), ",")[0]
		if !allowed[tag] {
			t.Fatalf("execution record holds an unexpected field %q; adapter data belongs in metadata", tag)
		}
	}

	implementer := &scriptedRunner{
		status: usableStatus(),
		result: RunResult{Status: RunStatusSuccess, SessionRef: "thread_1", Metadata: map[string]string{"sandbox": "workspace-write"}},
	}
	executor := Executor{
		Runner:      implementer,
		RunnerName:  "stub",
		Verifier:    &scriptedVerifier{outcomes: []VerificationOutcome{{Passed: true}}},
		Revisions:   &fixedRevisions{values: []string{"41a5f64", "9b2c1d0"}},
		MaxAttempts: 1,
		Now:         fixedNow,
	}

	record, err := executor.Execute(context.Background(), newRequest())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if record.Execution.BaseRevision != "41a5f64" || record.Execution.ResultRevision != "9b2c1d0" {
		t.Fatalf("execution = %+v", record.Execution)
	}
	if record.Execution.Metadata["sandbox"] != "workspace-write" {
		t.Fatalf("adapter data must live in metadata: %+v", record.Execution.Metadata)
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, exists := generic["sandbox"]; exists {
		t.Fatal("adapter data leaked to the record root")
	}
}

// P14: a fresh session resumes from durable state without session_ref.
func TestFreshSessionResumesWithoutSessionRef(t *testing.T) {
	record := ExecutionRecord{
		SchemaVersion: "0.1",
		Change:        "CH-100",
		Status:        OutcomeNeedsReview,
		Execution: ExecutionMeta{
			Runner:         "stub",
			SessionRef:     "thread_1",
			BaseRevision:   "41a5f64",
			ResultRevision: "9b2c1d0",
			Attempt:        2,
		},
	}

	record.Execution.SessionRef = ""

	execContext, err := NewExecutionContext(record)
	if err != nil {
		t.Fatalf("new execution context: %v", err)
	}
	if execContext.ChangeID != "CH-100" || execContext.BaseRevision != "41a5f64" || execContext.NextAttempt != 3 {
		t.Fatalf("context = %+v", execContext)
	}
	if execContext.SessionRef != "" {
		t.Fatalf("context must not invent a session ref: %+v", execContext)
	}

	if _, err := NewExecutionContext(ExecutionRecord{Execution: ExecutionMeta{BaseRevision: "x"}}); err == nil {
		t.Fatal("a record without a change must fail")
	}
}
