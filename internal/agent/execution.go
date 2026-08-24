package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RolePolicy is the ShipProof permission policy for one role. ShipProof owns
// the policy. An adapter translates it into the strongest mechanism that its
// runner supports.
type RolePolicy struct {
	Role           AgentRole
	WorkspaceWrite bool
	GitPush        bool
}

// PolicyFor returns the v0 policy for one role. Section 13.11 of the SDD sets
// these values. No role can push.
func PolicyFor(role AgentRole) (RolePolicy, error) {
	switch role {
	case RoleImplementer:
		return RolePolicy{Role: role, WorkspaceWrite: true, GitPush: false}, nil
	case RoleReviewer:
		return RolePolicy{Role: role, WorkspaceWrite: false, GitPush: false}, nil
	default:
		return RolePolicy{}, fmt.Errorf("unknown agent role %q", role)
	}
}

// Enforceable reports whether a runner can enforce the policy. ShipProof
// prefers explicit degradation over a prompt instruction presented as an
// enforcement boundary.
func (policy RolePolicy) Enforceable(capabilities RunnerCapabilities) (bool, string) {
	if policy.WorkspaceWrite {
		if !capabilities.WorkspaceWrite {
			return false, fmt.Sprintf("the runner cannot enforce workspace write for the %s role", policy.Role)
		}
		return true, ""
	}
	if !capabilities.ReadOnly {
		return false, fmt.Sprintf("the runner cannot enforce a read-only workspace for the %s role", policy.Role)
	}
	return true, ""
}

// Constraints returns the policy statements that an adapter passes to the
// runner in addition to its own enforcement mechanism.
func (policy RolePolicy) Constraints() []string {
	constraints := []string{
		"Keep every edit inside the approved change scope.",
		"Do not push to a remote.",
		"Do not weaken a test to make verification pass.",
	}
	if !policy.WorkspaceWrite {
		constraints = append([]string{"Do not modify any file. Report findings only."}, constraints...)
	}
	return constraints
}

// BuildPrompt renders the runner-neutral instruction text for one request.
// Prompt shape is ShipProof policy. Transport is an adapter detail.
func BuildPrompt(req RunRequest) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Role: %s\n", req.Role)
	fmt.Fprintf(&builder, "Change: %s\n", req.Change.ID)
	if strings.TrimSpace(req.Change.Title) != "" {
		fmt.Fprintf(&builder, "Title: %s\n", req.Change.Title)
	}
	if strings.TrimSpace(req.Change.SnapshotPath) != "" {
		fmt.Fprintf(&builder, "Intent snapshot: %s\n", req.Change.SnapshotPath)
	}
	builder.WriteString("\nTask:\n")
	builder.WriteString(strings.TrimSpace(req.Instructions))
	builder.WriteString("\n")
	if strings.TrimSpace(req.Change.Intent) != "" {
		builder.WriteString("\nApproved intent:\n")
		builder.WriteString(strings.TrimSpace(req.Change.Intent))
		builder.WriteString("\n")
	}
	if len(req.Constraints) > 0 {
		builder.WriteString("\nConstraints:\n")
		for _, constraint := range req.Constraints {
			fmt.Fprintf(&builder, "- %s\n", constraint)
		}
	}
	return builder.String()
}

// Outcome is the ShipProof decision for one `shipproof run`. It comes from
// deterministic verification, never from a runner claim.
type Outcome string

const (
	OutcomePass        Outcome = "PASS"
	OutcomeNeedsReview Outcome = "NEEDS_REVIEW"
	OutcomeBlocked     Outcome = "BLOCKED"
)

// VerificationOutcome is the deterministic verification result for one attempt.
type VerificationOutcome struct {
	Passed   bool
	ExitCode int
	Detail   string
}

// Verifier runs the deterministic verification plan for one change.
type Verifier interface {
	Verify(ctx context.Context, changeID string) (VerificationOutcome, error)
}

// RevisionSource reports the real Git state of the workspace. ShipProof
// inspects the repository itself. A runner claim is never evidence.
type RevisionSource interface {
	Revision(ctx context.Context) (string, error)
}

// Finding is one adversarial reviewer observation. Reviewer findings are
// agent-inferred. They must never be labeled as observed.
type Finding struct {
	Source     string `json:"source"`
	Summary    string `json:"summary"`
	Provenance string `json:"provenance"`
}

// ProvenanceInferred is the only provenance that a reviewer finding can carry.
const ProvenanceInferred = "inferred"

// AttemptRecord holds one implementer run and its verification result.
type AttemptRecord struct {
	Attempt      int    `json:"attempt"`
	RunnerStatus string `json:"runner_status"`
	Verification string `json:"verification"`
	ExitCode     int    `json:"exit_code"`
	Summary      string `json:"summary,omitempty"`
}

// ExecutionMeta is the runner-neutral execution header. Adapter-specific data
// belongs in Metadata and nowhere else.
type ExecutionMeta struct {
	Runner         string            `json:"runner"`
	RunnerVersion  string            `json:"runner_version,omitempty"`
	SessionRef     string            `json:"session_ref,omitempty"`
	StartedAt      string            `json:"started_at"`
	CompletedAt    string            `json:"completed_at"`
	BaseRevision   string            `json:"base_revision"`
	ResultRevision string            `json:"result_revision"`
	Attempt        int               `json:"attempt"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ExecutionRecord is the durable state of one `shipproof run`.
type ExecutionRecord struct {
	SchemaVersion string          `json:"schema_version"`
	Change        string          `json:"change"`
	Status        Outcome         `json:"status"`
	Detail        string          `json:"detail,omitempty"`
	Execution     ExecutionMeta   `json:"execution"`
	Attempts      []AttemptRecord `json:"attempts"`
	Findings      []Finding       `json:"findings,omitempty"`
}

// ExecutionContext is the input needed to continue work on a change. A fresh
// session must be able to build one from durable state alone, without
// session_ref.
type ExecutionContext struct {
	ChangeID     string
	BaseRevision string
	NextAttempt  int
	SessionRef   string
}

// NewExecutionContext builds a continuation context from durable state.
// It never requires session_ref.
func NewExecutionContext(record ExecutionRecord) (ExecutionContext, error) {
	if strings.TrimSpace(record.Change) == "" {
		return ExecutionContext{}, fmt.Errorf("execution record has no change")
	}
	if strings.TrimSpace(record.Execution.BaseRevision) == "" {
		return ExecutionContext{}, fmt.Errorf("execution record has no base revision")
	}
	return ExecutionContext{
		ChangeID:     record.Change,
		BaseRevision: record.Execution.BaseRevision,
		NextAttempt:  record.Execution.Attempt + 1,
		SessionRef:   record.Execution.SessionRef,
	}, nil
}

// Executor runs the SDD Section 13.9 execution flow for one change.
type Executor struct {
	Runner           AgentRunner
	RunnerName       string
	ReviewRunner     AgentRunner
	ReviewRunnerName string
	Verifier         Verifier
	Revisions        RevisionSource
	MaxAttempts      int
	Now              func() string
}

const executionSchemaVersion = "0.1"

func (executor Executor) now() string {
	if executor.Now != nil {
		return executor.Now()
	}
	return time.Now().UTC().Format(time.RFC3339)
}

// Execute runs the bounded implementation loop, then the adversarial review.
// It returns PASS, NEEDS_REVIEW, or BLOCKED. A runner claim never decides the
// outcome.
func (executor Executor) Execute(ctx context.Context, req RunRequest) (ExecutionRecord, error) {
	record := ExecutionRecord{
		SchemaVersion: executionSchemaVersion,
		Change:        req.Change.ID,
		Attempts:      []AttemptRecord{},
		Execution: ExecutionMeta{
			Runner:    executor.RunnerName,
			StartedAt: executor.now(),
			Metadata:  map[string]string{},
		},
	}

	maxAttempts := executor.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	if executor.Runner == nil || executor.Verifier == nil || executor.Revisions == nil {
		return record, fmt.Errorf("executor needs a runner, a verifier, and a revision source")
	}

	base, err := executor.Revisions.Revision(ctx)
	if err != nil {
		return record, fmt.Errorf("read base revision: %w", err)
	}
	record.Execution.BaseRevision = base
	record.Execution.ResultRevision = base

	status, err := executor.Runner.Probe(ctx)
	if err != nil {
		return executor.blocked(record, fmt.Sprintf("probe failed for runner %s", executor.RunnerName)), nil
	}
	record.Execution.RunnerVersion = status.Version
	if !status.Usable() {
		return executor.blocked(record, detailOr(status.Detail, "the runner is not installed or not authenticated")), nil
	}

	policy, err := PolicyFor(RoleImplementer)
	if err != nil {
		return record, err
	}
	if ok, detail := policy.Enforceable(status.Capabilities); !ok {
		return executor.blocked(record, detail), nil
	}

	implementRequest := req
	implementRequest.Role = RoleImplementer
	implementRequest.Constraints = policy.Constraints()

	verified := false
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		record.Execution.Attempt = attempt

		attemptRequest := implementRequest
		if attempt > 1 {
			attemptRequest.Instructions = repairInstructions(req.Instructions, record.Attempts[attempt-2])
		}

		result, err := executor.Runner.Run(ctx, attemptRequest)
		if err != nil {
			return executor.blocked(record, fmt.Sprintf("runner %s could not execute the change", executor.RunnerName)), nil
		}
		if result.SessionRef != "" {
			record.Execution.SessionRef = result.SessionRef
		}
		for key, value := range result.Metadata {
			record.Execution.Metadata[key] = value
		}

		revision, err := executor.Revisions.Revision(ctx)
		if err != nil {
			return record, fmt.Errorf("read result revision: %w", err)
		}
		record.Execution.ResultRevision = revision

		outcome, err := executor.Verifier.Verify(ctx, req.Change.ID)
		if err != nil {
			return record, fmt.Errorf("run verification: %w", err)
		}

		attemptRecord := AttemptRecord{
			Attempt:      attempt,
			RunnerStatus: string(result.Status),
			Verification: verificationLabel(outcome.Passed),
			ExitCode:     outcome.ExitCode,
			Summary:      outcome.Detail,
		}
		record.Attempts = append(record.Attempts, attemptRecord)

		if outcome.Passed {
			verified = true
			break
		}
	}

	if !verified {
		record.Status = OutcomeNeedsReview
		record.Detail = fmt.Sprintf("verification failed after %d attempt(s)", len(record.Attempts))
		record.Execution.CompletedAt = executor.now()
		return record, nil
	}

	findings, blockedDetail, err := executor.review(ctx, req)
	if err != nil {
		return record, err
	}
	if blockedDetail != "" {
		return executor.blocked(record, blockedDetail), nil
	}
	record.Findings = findings

	record.Status = OutcomePass
	record.Execution.CompletedAt = executor.now()
	return record, nil
}

// review runs the adversarial reviewer. Findings enter the record as
// agent-inferred.
func (executor Executor) review(ctx context.Context, req RunRequest) ([]Finding, string, error) {
	runner := executor.ReviewRunner
	name := executor.ReviewRunnerName
	if runner == nil {
		return nil, "", nil
	}
	if name == "" {
		name = executor.RunnerName
	}

	status, err := runner.Probe(ctx)
	if err != nil {
		return nil, fmt.Sprintf("probe failed for review runner %s", name), nil
	}
	if !status.Usable() {
		return nil, detailOr(status.Detail, fmt.Sprintf("review runner %s is not usable", name)), nil
	}

	policy, err := PolicyFor(RoleReviewer)
	if err != nil {
		return nil, "", err
	}
	if ok, detail := policy.Enforceable(status.Capabilities); !ok {
		return nil, detail, nil
	}

	reviewRequest := req
	reviewRequest.Role = RoleReviewer
	reviewRequest.Constraints = policy.Constraints()
	reviewRequest.Instructions = "Review the implemented change against the approved intent. Report material defects only."

	result, err := runner.Run(ctx, reviewRequest)
	if err != nil {
		return nil, fmt.Sprintf("review runner %s could not execute", name), nil
	}

	return parseFindings(name, result.Summary), "", nil
}

// parseFindings converts reviewer prose into agent-inferred findings.
// ShipProof never presents a reviewer claim as an observed fact.
func parseFindings(source, summary string) []Finding {
	var findings []Finding
	for _, line := range strings.Split(summary, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		if text == "" {
			continue
		}
		findings = append(findings, Finding{Source: source, Summary: text, Provenance: ProvenanceInferred})
	}
	if len(findings) == 0 && strings.TrimSpace(summary) != "" {
		findings = append(findings, Finding{Source: source, Summary: strings.TrimSpace(summary), Provenance: ProvenanceInferred})
	}
	return findings
}

func (executor Executor) blocked(record ExecutionRecord, detail string) ExecutionRecord {
	record.Status = OutcomeBlocked
	record.Detail = detail
	record.Execution.CompletedAt = executor.now()
	return record
}

func repairInstructions(original string, previous AttemptRecord) string {
	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(original))
	builder.WriteString("\n\nThe previous attempt did not pass verification. ")
	builder.WriteString("Repair the change inside the approved scope. Do not widen the scope.\n")
	if strings.TrimSpace(previous.Summary) != "" {
		builder.WriteString("\nVerification detail:\n")
		builder.WriteString(strings.TrimSpace(previous.Summary))
		builder.WriteString("\n")
	}
	return builder.String()
}

func verificationLabel(passed bool) string {
	if passed {
		return "pass"
	}
	return "fail"
}

func detailOr(detail, fallback string) string {
	if strings.TrimSpace(detail) != "" {
		return detail
	}
	return fallback
}
