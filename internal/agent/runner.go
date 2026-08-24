package agent

import "context"

// AgentRole names the policy role that a run executes under. Section 13.11 of
// the v0 SDD binds each role to a permission policy.
type AgentRole string

const (
	RoleImplementer AgentRole = "implementer"
	RoleReviewer    AgentRole = "reviewer"
)

// RunStatus is the runner-reported outcome of one bounded coding task. A
// runner claim is never evidence. ShipProof decides the change outcome from
// deterministic verification.
type RunStatus string

const (
	RunStatusSuccess RunStatus = "success"
	RunStatusFailed  RunStatus = "failed"
	RunStatusBlocked RunStatus = "blocked"
)

// Change carries the runner-neutral change facts that a runner needs. It holds
// no adapter concept and no Linear concept.
type Change struct {
	ID           string
	Title        string
	SnapshotPath string
	Intent       string
}

// RunRequest is the complete input for one bounded runner execution.
type RunRequest struct {
	Workspace    string
	Change       Change
	Role         AgentRole
	Instructions string
	Constraints  []string
}

// RunResult is the complete output of one bounded runner execution.
// SessionRef is opaque. ShipProof must not depend on its syntax.
type RunResult struct {
	Status     RunStatus
	Summary    string
	SessionRef string
	Metadata   map[string]string
}

// AgentRunner is the only interface that ShipProof models for an execution
// runtime. It exposes Probe and Run and nothing else.
type AgentRunner interface {
	Probe(ctx context.Context) (RunnerStatus, error)
	Run(ctx context.Context, req RunRequest) (RunResult, error)
}
