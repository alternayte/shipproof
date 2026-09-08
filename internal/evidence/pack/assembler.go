package pack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alternayte/shipproof/internal/agent"
	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/evidence"
	"github.com/alternayte/shipproof/internal/git"
	"github.com/alternayte/shipproof/internal/grade"
	"github.com/alternayte/shipproof/internal/phase"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/schema"
	"github.com/alternayte/shipproof/internal/verdict"
	"github.com/alternayte/shipproof/internal/verification"
	"github.com/alternayte/shipproof/internal/version"
)

type Options struct {
	EvidenceFiles []string
	BaseRev       string
	HeadRev       string
	// Warn receives one line for a section the pack omits. It is nil when
	// the caller wants no report. Silence is the wrong default for a signal
	// whose purpose is to be read.
	Warn io.Writer
}

func Assemble(root, changeID string, opts Options) (schema.EvidencePack, error) {
	pack := schema.EvidencePack{
		SchemaVersion: schema.CurrentVersion,
		ChangeID:      changeID,
		Checks:        []schema.Check{},
		Requirements:  []schema.RequirementRow{},
		EmptySections: map[string]string{},
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

	pack.Intent = buildIntent(record)

	if stale, err := record.Staleness(root); err != nil {
		pack.Intent.Stale = true
	} else {
		pack.Intent.Stale = stale.Stale
		pack.Intent.CurrentSourceHash = stale.CurrentHash
	}
	if pack.Intent.Stale {
		pack.Checks = append(pack.Checks, schema.Check{
			ID:         "intent:staleness",
			Status:     "fail",
			Source:     "shipproof",
			Provenance: schema.ProvenanceDerived,
		})
	} else {
		pack.Checks = append(pack.Checks, schema.Check{
			ID:         "intent:staleness",
			Status:     "pass",
			Source:     "shipproof",
			Provenance: schema.ProvenanceDerived,
		})
	}

	runChecks, err := loadRunChecks(root, changeID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return pack, fmt.Errorf("load run result: %w", err)
	}
	pack.Checks = append(pack.Checks, runChecks...)

	evChecks, err := evidence.ParseFiles(opts.EvidenceFiles)
	if err != nil {
		return pack, fmt.Errorf("parse evidence files: %w", err)
	}
	pack.Checks = append(pack.Checks, evChecks...)

	// SDD Section 11.6: the coverage signal never blocks a run and never
	// fails a change. A malformed sidecar or a malformed proof-results file
	// adds no coverage check, the same way buildUnexplained adds no finding
	// when its own inputs are unreadable.
	if requirements.Exists(root, changeID) {
		if matrix, err := coverage.Read(root, changeID, plan); err == nil {
			pack.Checks = append(pack.Checks, coverageChecks(matrix)...)
		}
	}

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
			pack.Checks = append(pack.Checks, gitCheck)
		} else {
			pack.Implementation = implementationFromGit(gitMeta)
			gitCheck := schema.Check{
				ID:         "git:collect",
				Status:     "pass",
				Source:     "git",
				Provenance: schema.ProvenanceObserved,
			}
			pack.Checks = append(pack.Checks, gitCheck)
		}
	}

	// Section 11.6: the unexplained-change signal is review material, and a
	// reader must know when it is absent. The documented flow passes no
	// --base, so the recorded base revision stands in for it.
	base, baseSource := resolveBase(root, changeID, opts.BaseRev)
	if base == "" {
		reason := "no base revision is known. Pass --base <rev>."
		pack.EmptySections["unexplained_change"] = reason
		warn(opts.Warn, "unexplained change: the section is empty, because "+reason)
	} else if measured := buildUnexplained(root, changeID, plan, base, head); measured != nil {
		pack.UnexplainedChange = *measured
		pack.UnexplainedChange.Measured = true
	} else {
		reason := fmt.Sprintf("git could not read the range %s..%s from %s.", base, head, baseSource)
		pack.EmptySections["unexplained_change"] = reason
		warn(opts.Warn, "unexplained change: the section is empty, because "+reason)
	}

	if agentRun, err := loadAgentRun(root, changeID); err == nil {
		pack.Agent = agentRun
	} else {
		pack.EmptySections["agent"] = "no telemetry record exists for this change."
	}

	// Section 7 names no agent-review section. A reviewer finding is an agent
	// claim, and a claim belongs in the check list with a claimed grade.
	if execution, err := agent.LoadExecution(root, changeID); err == nil {
		for _, finding := range execution.Findings {
			pack.Checks = append(pack.Checks, schema.Check{
				ID:         "agent:review:" + finding.Source,
				Status:     "unknown",
				Source:     execution.Execution.Runner,
				Provenance: schema.ProvenanceInferred,
				Detail:     finding.Summary,
			})
		}
	}

	pack.Requirements = requirementRows(root, changeID, plan)

	// Decision D2 of Section 15. A local pack stays unsigned. Step 5 of the
	// sequence fills this section in continuous integration.
	pack.EmptySections["attestation"] = "a local pack is unsigned. Only a build-system pack is audit-grade."

	pack.Verdict = buildVerdict(root, pack)

	statePackReasons(&pack)

	pack.Provenance = schema.PackProvenance{
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
		ShipProofVersion: version.Version,
	}

	if err := pack.Validate(); err != nil {
		return pack, fmt.Errorf("validate pack: %w", err)
	}

	return pack, nil
}

// resolveBase picks the base revision for the changed-line diff. The option
// wins. The agent execution record stands in when the option is empty. The
// second result names the source, for the message that reports a failure.
func resolveBase(root, changeID, option string) (string, string) {
	if base := strings.TrimSpace(option); base != "" {
		return base, "the --base option"
	}
	execution, err := agent.LoadExecution(root, changeID)
	if err != nil {
		return "", ""
	}
	return strings.TrimSpace(execution.Execution.BaseRevision), "the recorded agent run"
}

// warn writes one line for an omitted section. A nil writer reports nothing.
func warn(writer io.Writer, message string) {
	if writer == nil {
		return
	}
	fmt.Fprintln(writer, message)
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

func buildIntent(record change.Record) schema.IntentEvidence {
	return schema.IntentEvidence{
		SourcePath:   record.SourcePath,
		SnapshotHash: record.SHA256,
		CapturedAt:   record.CapturedAt,
	}
}

// requirementRows builds one row per requirement, with its proof commands and
// its proof result. The verification plan is the spine, because the plan names
// every requirement that this change must prove. The coverage matrix supplies
// the state when a requirement sidecar exists. A requirement that the sidecar
// names and the plan does not is appended, so no requirement disappears.
func requirementRows(root, changeID string, plan verification.Plan) []schema.RequirementRow {
	rows := []schema.RequirementRow{}

	var order []string
	commands := map[string][]string{}
	for _, group := range [][]verification.Item{plan.Requirements, plan.Invariants} {
		for _, item := range group {
			order = append(order, item.ID)
			commands[item.ID] = proofCommands(item.Proof)
		}
	}

	judged := map[string]coverage.Row{}
	var matrixOrder []string
	if requirements.Exists(root, changeID) {
		if matrix, err := coverage.Read(root, changeID, plan); err == nil {
			for _, row := range matrix.Rows {
				judged[row.RequirementID] = row
				matrixOrder = append(matrixOrder, row.RequirementID)
			}
		}
	}

	seen := map[string]bool{}
	appendRow := func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		row := schema.RequirementRow{
			ID:        id,
			ProofRefs: commands[id],
			State:     string(coverage.Unproven),
			Grade:     string(grade.Claimed),
			Detail:    "no requirement sidecar records a result for this requirement",
		}
		if found, ok := judged[id]; ok {
			row.Statement = found.Statement
			row.State = string(found.State)
			row.Grade = string(grade.FromProvenance(checkProvenance(found.Provenance)))
			row.Detail = found.Detail
		}
		rows = append(rows, row)
	}

	for _, id := range order {
		appendRow(id)
	}
	for _, id := range matrixOrder {
		appendRow(id)
	}
	return rows
}

// buildVerdict states the outcome of Section 5 from the pack that this run
// assembled. A pack that states no verdict answers nothing.
func buildVerdict(root string, pack schema.EvidencePack) schema.VerdictEvidence {
	matrix := coverage.Matrix{ChangeID: pack.ChangeID}
	for _, row := range pack.Requirements {
		matrix.Rows = append(matrix.Rows, coverage.Row{
			RequirementID: row.ID,
			State:         coverage.State(row.State),
		})
	}

	var lines *int
	if pack.UnexplainedChange.Measured {
		total := 0
		for _, finding := range pack.UnexplainedChange.LineFindings {
			span := finding.EndLine - finding.StartLine + 1
			if span < 1 {
				span = 1
			}
			total += span
		}
		lines = &total
	}

	next := phase.Result{ChangeID: pack.ChangeID}
	if resolved, err := phase.Resolve(root, pack.ChangeID); err == nil {
		next = resolved
	}

	block := verdict.Decide(verdict.Input{
		ChangeID:    pack.ChangeID,
		Phase:       next,
		Matrix:      matrix,
		HasMatrix:   len(matrix.Rows) > 0,
		Unexplained: lines,
		Checks:      pack.Checks,
	})
	return schema.VerdictEvidence{Verdict: string(block.Verdict), Reason: block.Reason, Next: block.Next}
}

// statePackReasons fills every remaining entry of empty_sections. Rule 2 of
// Section 7 forbids a silent omission.
func statePackReasons(pack *schema.EvidencePack) {
	if len(pack.Requirements) == 0 {
		setReason(pack, "requirements", "the change holds no requirement set, so no requirement has a recorded result.")
	}
	if len(pack.Checks) == 0 {
		setReason(pack, "checks", "no tool result and no human acceptance reached this pack.")
	}
	if len(pack.Implementation.Commits) == 0 && len(pack.Implementation.ChangedFiles) == 0 {
		setReason(pack, "implementation", "no base revision is known, so git reported no commit and no changed file.")
	}
	if !pack.UnexplainedChange.Measured {
		setReason(pack, "unexplained_change", "ShipProof did not reach the measurement.")
	}
	if pack.Agent == nil {
		setReason(pack, "agent", "no telemetry record exists for this change.")
	}
	if pack.Attestation == nil {
		setReason(pack, "attestation", "a local pack is unsigned. Only a build-system pack is audit-grade.")
	}
}

// setReason keeps the first reason. An earlier stage knows the exact cause,
// and the fallback knows only the category.
func setReason(pack *schema.EvidencePack, name, reason string) {
	if _, stated := pack.EmptySections[name]; stated {
		return
	}
	pack.EmptySections[name] = reason
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

func loadAgentRun(root, changeID string) (*schema.AgentEvidence, error) {
	runPath := filepath.Join(root, ".shipproof", "runs", changeID, "agent-run.json")
	data, err := os.ReadFile(runPath)
	if err != nil {
		return nil, err
	}

	var run schema.AgentEvidence
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parse agent run record: %w", err)
	}

	return &run, nil
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
