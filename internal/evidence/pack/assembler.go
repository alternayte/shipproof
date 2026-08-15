package pack

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shipproof/shipproof/internal/change"
	"github.com/shipproof/shipproof/internal/evidence"
	"github.com/shipproof/shipproof/internal/git"
	"github.com/shipproof/shipproof/internal/schema"
	"github.com/shipproof/shipproof/internal/shaping"
	"github.com/shipproof/shipproof/internal/verification"
)

type Options struct {
	EvidenceFiles []string
	BaseRev       string
	HeadRev       string
}

func Assemble(root, changeID string, opts Options) (schema.EvidencePack, error) {
	pack := schema.EvidencePack{
		SchemaVersion: schema.CurrentVersion,
		ChangeID:      changeID,
		Verification: schema.VerificationEvidence{
			Checks: []schema.Check{},
		},
	}

	record, err := change.Load(root, changeID)
	if err != nil {
		return pack, fmt.Errorf("load change record: %w", err)
	}

	planPath := verification.Path(root, changeID)
	plan, err := verification.Load(planPath)
	if err != nil {
		return pack, fmt.Errorf("load verification plan: %w", err)
	}

	pack.Intent = buildIntent(record, plan)

	runChecks, err := loadRunChecks(root, changeID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return pack, fmt.Errorf("load run result: %w", err)
	}
	pack.Verification.Checks = append(pack.Verification.Checks, runChecks...)

	evChecks, err := evidence.ParseFiles(opts.EvidenceFiles)
	if err != nil {
		return pack, fmt.Errorf("parse evidence files: %w", err)
	}
	pack.Verification.Checks = append(pack.Verification.Checks, evChecks...)

	head := opts.HeadRev
	if head == "" {
		head = "HEAD"
	}
	if opts.BaseRev != "" {
		gitMeta, err := git.CollectMetadata(root, opts.BaseRev, head)
		if err != nil {
			gitCheck := schema.Check{
				ID:         "git:collect",
				Status:     "fail",
				Source:     "git",
				Provenance: schema.ProvenanceObserved,
			}
			pack.Verification.Checks = append(pack.Verification.Checks, gitCheck)
		} else {
			pack.Implementation = implementationFromGit(gitMeta)
			gitCheck := schema.Check{
				ID:         "git:collect",
				Status:     "pass",
				Source:     "git",
				Provenance: schema.ProvenanceObserved,
			}
			pack.Verification.Checks = append(pack.Verification.Checks, gitCheck)
		}
	}

	if agentRun, err := loadAgentRun(root, changeID); err == nil {
		pack.AgentRun = agentRun
	}

	pack.Readiness = loadReadiness(root, record)

	review, err := loadReview(root, changeID)
	if err != nil {
		return pack, err
	}
	if review != nil {
		pack.Review = review
		pack.Verification.Checks = append(pack.Verification.Checks, schema.Check{
			ID:         "github:review",
			Status:     "pass",
			Source:     "github",
			Provenance: schema.ProvenanceObserved,
		})
	}

	pack.Provenance = schema.PackProvenance{
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		ShipProofVersion: schema.CurrentVersion,
	}

	if err := pack.Validate(); err != nil {
		return pack, fmt.Errorf("validate pack: %w", err)
	}

	return pack, nil
}

func WritePack(root string, pack schema.EvidencePack) error {
	if err := pack.Validate(); err != nil {
		return fmt.Errorf("validate pack before write: %w", err)
	}

	dir := filepath.Join(root, ".shipproof", "changes", pack.ChangeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create change directory: %w", err)
	}

	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return fmt.Errorf("encode evidence pack: %w", err)
	}
	data = append(data, '\n')

	packPath := filepath.Join(dir, "evidence-pack.json")
	if err := os.WriteFile(packPath, data, 0o644); err != nil {
		return fmt.Errorf("write evidence pack: %w", err)
	}

	return nil
}

func buildIntent(record change.Record, plan verification.Plan) schema.IntentEvidence {
	var requirements []schema.Requirement
	for _, item := range plan.Requirements {
		requirements = append(requirements, schema.Requirement{
			ID:               item.ID,
			VerificationRefs: proofCommands(item.Proof),
		})
	}
	for _, item := range plan.Invariants {
		requirements = append(requirements, schema.Requirement{
			ID:               item.ID,
			VerificationRefs: proofCommands(item.Proof),
		})
	}

	return schema.IntentEvidence{
		SnapshotHash: record.SHA256,
		Requirements: requirements,
	}
}

func proofCommands(proofs []verification.Proof) []string {
	var commands []string
	for _, p := range proofs {
		if p.Command != "" {
			commands = append(commands, p.Command)
		}
	}
	return commands
}

func loadRunChecks(root, changeID string) ([]schema.Check, error) {
	runPath := filepath.Join(root, ".shipproof", "runs", changeID, "run.json")
	data, err := os.ReadFile(runPath)
	if err != nil {
		return nil, err
	}

	var result struct {
		ExitCode   int    `json:"exit_code"`
		DurationMs int64  `json:"duration_ms"`
		Timestamp  string `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse run result: %w", err)
	}

	status := "pass"
	if result.ExitCode != 0 {
		status = "fail"
	}

	return []schema.Check{
		{
			ID:         "verification:run",
			Status:     status,
			Source:     "shipproof-runner",
			Provenance: schema.ProvenanceObserved,
		},
	}, nil
}

func loadAgentRun(root, changeID string) (*schema.AgentRunMetadata, error) {
	runPath := filepath.Join(root, ".shipproof", "runs", changeID, "agent-run.json")
	data, err := os.ReadFile(runPath)
	if err != nil {
		return nil, err
	}

	var run schema.AgentRunMetadata
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parse agent run record: %w", err)
	}

	return &run, nil
}

func loadReadiness(root string, record change.Record) *schema.ReadinessEvidence {
	if record.ShapingRef == "" {
		return nil
	}

	session, _, err := shaping.Load(root, record.ShapingRef)
	if err != nil {
		return nil
	}

	return &schema.ReadinessEvidence{
		ShapingRef:   record.ShapingRef,
		BlockerCount: len(session.Readiness.Blockers),
	}
}

func loadReview(root, changeID string) (*schema.ReviewEvidence, error) {
	reviewPath := filepath.Join(root, ".shipproof", "changes", changeID, "review.json")
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read review.json: %w", err)
	}

	var review schema.ReviewEvidence
	if err := json.Unmarshal(data, &review); err != nil {
		return nil, fmt.Errorf("parse review.json: %w", err)
	}

	return &review, nil
}

func implementationFromGit(meta git.Metadata) schema.ImplementationEvidence {
	var commits []schema.ImplementationCommit
	for _, c := range meta.Commits {
		commits = append(commits, schema.ImplementationCommit{
			Hash:      c.Hash,
			Author:    c.Author,
			Timestamp: c.Timestamp,
			Subject:   c.Subject,
		})
	}

	return schema.ImplementationEvidence{
		Commits:      commits,
		ChangedFiles: meta.ChangedFiles,
		Additions:    meta.Additions,
		Deletions:    meta.Deletions,
		DiffStat:     meta.DiffStat,
	}
}
