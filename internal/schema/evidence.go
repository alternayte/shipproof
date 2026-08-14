package schema

import (
	"errors"
	"fmt"
)

type ProvenanceKind string

const (
	ProvenanceObserved ProvenanceKind = "observed"
	ProvenanceDerived  ProvenanceKind = "derived"
	ProvenanceInferred ProvenanceKind = "inferred"
	ProvenanceHuman    ProvenanceKind = "human"
)

type EvidencePack struct {
	SchemaVersion  string                 `json:"schema_version"`
	ChangeID       string                 `json:"change_id"`
	Intent         IntentEvidence         `json:"intent"`
	Implementation ImplementationEvidence `json:"implementation"`
	Verification   VerificationEvidence   `json:"verification"`
	Provenance     PackProvenance         `json:"provenance"`
	AgentRun       *AgentRunMetadata      `json:"agent_run,omitempty"`
}

type AgentRunMetadata struct {
	Provider      string          `json:"provider,omitempty"`
	AgentVersion  string          `json:"agent_version,omitempty"`
	Model         string          `json:"model,omitempty"`
	StartedAt     string          `json:"started_at,omitempty"`
	EndedAt       string          `json:"ended_at,omitempty"`
	SessionID     string          `json:"session_id,omitempty"`
	Cost          float64         `json:"cost,omitempty"`
	Tokens        *TokenUsageMeta `json:"tokens,omitempty"`
	ToolCallCount int64           `json:"tool_call_count,omitempty"`
	ExitStatus    string          `json:"exit_status,omitempty"`
	RawLogRef     string          `json:"raw_log_ref,omitempty"`
}

type TokenUsageMeta struct {
	Input  int64 `json:"input,omitempty"`
	Output int64 `json:"output,omitempty"`
}

type ImplementationEvidence struct {
	Commits      []ImplementationCommit `json:"commits"`
	ChangedFiles []string               `json:"changed_files"`
	Additions    int                    `json:"additions"`
	Deletions    int                    `json:"deletions"`
	DiffStat     string                 `json:"diff_stat"`
}

type ImplementationCommit struct {
	Hash      string `json:"hash"`
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
	Subject   string `json:"subject"`
}

type IntentEvidence struct {
	SnapshotHash string        `json:"snapshot_hash"`
	Requirements []Requirement `json:"requirements"`
}

type Requirement struct {
	ID               string   `json:"id"`
	VerificationRefs []string `json:"verification_refs,omitempty"`
}

type VerificationEvidence struct {
	Checks []Check `json:"checks"`
}

type Check struct {
	ID         string         `json:"id"`
	Status     string         `json:"status"`
	Source     string         `json:"source"`
	Provenance ProvenanceKind `json:"provenance"`
}

type PackProvenance struct {
	GeneratedAt      string `json:"generated_at"`
	ShipProofVersion string `json:"shipproof_version"`
}

func (pack EvidencePack) Validate() error {
	if pack.SchemaVersion != CurrentVersion {
		return fmt.Errorf("schema_version must be %q", CurrentVersion)
	}
	if pack.ChangeID == "" {
		return errors.New("change_id is required")
	}
	if pack.Intent.SnapshotHash == "" {
		return errors.New("intent.snapshot_hash is required")
	}
	if pack.Provenance.GeneratedAt == "" {
		return errors.New("provenance.generated_at is required")
	}
	if pack.Provenance.ShipProofVersion == "" {
		return errors.New("provenance.shipproof_version is required")
	}

	for index, check := range pack.Verification.Checks {
		if check.ID == "" {
			return fmt.Errorf("verification.checks[%d].id is required", index)
		}
		switch check.Status {
		case "pass", "fail", "skip", "unknown":
		default:
			return fmt.Errorf("verification.checks[%d].status is invalid", index)
		}
		switch check.Provenance {
		case ProvenanceObserved, ProvenanceDerived, ProvenanceInferred, ProvenanceHuman:
		default:
			return fmt.Errorf("verification.checks[%d].provenance is invalid", index)
		}
	}

	return nil
}
