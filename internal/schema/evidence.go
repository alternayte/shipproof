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
