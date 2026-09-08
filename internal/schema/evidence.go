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

// EvidencePack is the single published artifact. Section 7 of the design
// document names every top-level field. A pack is complete or absent, and it
// states why a section is empty. It never omits a section in silence.
type EvidencePack struct {
	SchemaVersion     string                 `json:"schema_version"`
	ChangeID          string                 `json:"change_id"`
	Verdict           VerdictEvidence        `json:"verdict"`
	Intent            IntentEvidence         `json:"intent"`
	Implementation    ImplementationEvidence `json:"implementation"`
	Requirements      []RequirementRow       `json:"requirements"`
	Checks            []Check                `json:"checks"`
	UnexplainedChange UnexplainedEvidence    `json:"unexplained_change"`
	Agent             *AgentEvidence         `json:"agent"`
	Attestation       *AttestationEvidence   `json:"attestation"`
	// EmptySections maps the name of an empty or absent section onto the
	// reason. Rule 2 of Section 7 requires it. A reader must never mistake an
	// absent measurement for a measurement of zero.
	EmptySections map[string]string `json:"empty_sections"`
	Provenance    PackProvenance    `json:"provenance"`
}

// VerdictEvidence holds the three-line block of Section 5.
type VerdictEvidence struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
	Next    string `json:"next"`
}

// RequirementRow states what the artifacts say about one requirement. Grade
// holds one of the three grades of Section 6.
type RequirementRow struct {
	ID        string   `json:"id"`
	Statement string   `json:"statement,omitempty"`
	ProofRefs []string `json:"proof_refs,omitempty"`
	State     string   `json:"state"`
	Grade     string   `json:"grade"`
	Detail    string   `json:"detail,omitempty"`
}

// AttestationEvidence holds the signature block. A local pack carries none,
// and EmptySections then states that a local pack is unsigned.
type AttestationEvidence struct {
	Format      string `json:"format"`
	PayloadType string `json:"payload_type,omitempty"`
	// Signature is the base64 signature over the SHA-256 of the canonical
	// payload. The canonical payload excludes this block.
	Signature string `json:"signature"`
	// Certificate is the PEM signing certificate that the build system used.
	// The verifier reads the public key from it.
	Certificate string `json:"certificate,omitempty"`
	// Subject names the head revision that the attestation covers.
	Subject string `json:"subject,omitempty"`
	// Digest is the SHA-256 of the canonical payload, in hexadecimal.
	Digest string `json:"digest,omitempty"`
}

// UnexplainedEvidence records which changed code no approved proof ran. The
// line-level findings are observed. The file-level findings are claimed. The
// section never fails a change.
//
// Measured reports whether ShipProof reached the measurement at all. A false
// value means that the counts state nothing. It never means zero.
type UnexplainedEvidence struct {
	Measured            bool              `json:"measured"`
	CoverageAvailable   bool              `json:"coverage_available"`
	LineFindings        []UnexplainedLine `json:"line_findings"`
	FileFindings        []UnexplainedFile `json:"file_findings"`
	UninstrumentedLines int               `json:"uninstrumented_lines"`
}

type UnexplainedLine struct {
	File      string `json:"file"`
	Symbol    string `json:"symbol,omitempty"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type UnexplainedFile struct {
	Path          string `json:"path"`
	IgnorePattern string `json:"ignore_pattern,omitempty"`
}

// AgentEvidence names the model that produced the change and the session it
// ran under. A field that no tool reported stays missing.
type AgentEvidence struct {
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
	// SourcePath names the intent document that `start` snapshotted.
	SourcePath   string `json:"source_path,omitempty"`
	SnapshotHash string `json:"snapshot_hash"`
	// CapturedAt states when `start` took the snapshot.
	CapturedAt string `json:"captured_at,omitempty"`
	// Stale is true when the current source document differs from the
	// snapshot taken when implementation began. Stale evidence needs
	// re-verification against the current intent.
	Stale bool `json:"stale"`
	// CurrentSourceHash is the SHA-256 of the current source document.
	// It is empty when the source document is missing.
	CurrentSourceHash string `json:"current_source_hash,omitempty"`
}

type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Source string `json:"source"`
	// Grade is one of the three grades of Section 6. It answers the audit
	// question "Is the result machine-observed or asserted?" without an
	// external lookup table.
	Grade string `json:"grade"`
	// Provenance is the finer machine label that Section 6 permits. Grade
	// collapses it to three values for a reader.
	Provenance ProvenanceKind `json:"provenance"`
	// Detail states the reason for the status in one sentence. It is review
	// material, and its absence never changes the status it explains.
	Detail string `json:"detail,omitempty"`
}

type PackProvenance struct {
	GeneratedAt      string `json:"generated_at"`
	ShipProofVersion string `json:"shipproof_version"`
}

// Validate reports the first rule that a pack breaks. A pack that fails this
// check is never written. Rule 1 of Section 7 makes a partial pack a defect.
func (pack EvidencePack) Validate() error {
	if pack.SchemaVersion != CurrentVersion {
		return fmt.Errorf("schema_version must be %q", CurrentVersion)
	}
	if pack.ChangeID == "" {
		return errors.New("change_id is required")
	}
	if pack.Verdict.Verdict == "" {
		return errors.New("verdict.verdict is required")
	}
	if pack.Verdict.Reason == "" {
		return errors.New("verdict.reason is required")
	}
	if pack.Verdict.Next == "" {
		return errors.New("verdict.next is required")
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
	if pack.EmptySections == nil {
		return errors.New("empty_sections is required")
	}

	for index, check := range pack.Checks {
		if check.ID == "" {
			return fmt.Errorf("checks[%d].id is required", index)
		}
		switch check.Status {
		case "pass", "fail", "skip", "unknown":
		default:
			return fmt.Errorf("checks[%d].status is invalid", index)
		}
		switch check.Provenance {
		case ProvenanceObserved, ProvenanceDerived, ProvenanceInferred, ProvenanceHuman:
		default:
			return fmt.Errorf("checks[%d].provenance is invalid", index)
		}
		switch check.Grade {
		case "observed", "stated", "claimed":
		default:
			return fmt.Errorf("checks[%d].grade is invalid", index)
		}
	}

	for index, row := range pack.Requirements {
		if row.ID == "" {
			return fmt.Errorf("requirements[%d].id is required", index)
		}
		if row.State == "" {
			return fmt.Errorf("requirements[%d].state is required", index)
		}
		if row.Grade == "" {
			return fmt.Errorf("requirements[%d].grade is required", index)
		}
	}

	// Rule 2 of Section 7. Every empty section names its reason.
	for name, empty := range map[string]bool{
		"requirements":       len(pack.Requirements) == 0,
		"checks":             len(pack.Checks) == 0,
		"implementation":     len(pack.Implementation.Commits) == 0 && len(pack.Implementation.ChangedFiles) == 0,
		"unexplained_change": !pack.UnexplainedChange.Measured,
		"agent":              pack.Agent == nil,
		"attestation":        pack.Attestation == nil,
	} {
		if !empty {
			// empty_sections describes what is empty. A reason that survives
			// after a section is filled tells a reader the opposite of the
			// truth, so it is a defect and not a leftover.
			if _, stated := pack.EmptySections[name]; stated {
				return fmt.Errorf("the section %q holds content and empty_sections still calls it empty", name)
			}
			continue
		}
		if reason := pack.EmptySections[name]; reason == "" {
			return fmt.Errorf("the section %q is empty and empty_sections states no reason", name)
		}
	}

	return nil
}
