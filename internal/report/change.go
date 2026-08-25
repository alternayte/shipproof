package report

import (
	"github.com/alternayte/shipproof/internal/review"
	"github.com/alternayte/shipproof/internal/schema"
)

type changeReportData struct {
	ChangeID    string
	GeneratedAt string
	Intent      intentData
	Verify      verifyData
	Implement   implementData
	AgentRun    agentRunData
	Provenance  reportProvenanceData
	Unexplained unexplainedData
}

type unexplainedData struct {
	Present             bool
	CoverageAvailable   bool
	LineFindings        []schema.UnexplainedLine
	FileFindings        []schema.UnexplainedFile
	UninstrumentedLines int
}

type intentData struct {
	SnapshotHash     string
	RequirementCount int
	Provenance       string
}

type verifyData struct {
	Checks            []checkRow
	ProvenCheckCount  int
	FailCheckCount    int
	SkipCheckCount    int
	UnknownCheckCount int
	TotalChecks       int
}

type checkRow struct {
	ID         string
	Status     string
	Source     string
	Provenance schema.ProvenanceKind
}

type implementData struct {
	CommitCount      int
	ChangedFileCount int
	Additions        int
	Deletions        int
	DiffStat         string
	Commits          []schema.ImplementationCommit
	ChangedFiles     []string
}

type agentRunData struct {
	Hides         bool
	Provider      string
	Model         string
	SessionID     string
	Cost          float64
	InputTokens   int64
	OutputTokens  int64
	ToolCallCount int64
	StartedAt     string
	EndedAt       string
	ExitStatus    string
}

type reportProvenanceData struct {
	ShipProofVersion string
}

func buildIntentData(pack schema.EvidencePack) intentData {
	return intentData{
		SnapshotHash:     pack.Intent.SnapshotHash,
		RequirementCount: len(pack.Intent.Requirements),
		Provenance:       string(schema.ProvenanceObserved),
	}
}

func buildVerifyData(pack schema.EvidencePack, packet *review.ReviewPacket) verifyData {
	data := verifyData{}

	var passCount, failCount, skipCount, unknownCount int
	for _, check := range pack.Verification.Checks {
		data.Checks = append(data.Checks, checkRow{
			ID:         check.ID,
			Status:     check.Status,
			Source:     check.Source,
			Provenance: check.Provenance,
		})
		switch check.Status {
		case "pass":
			passCount++
		case "fail":
			failCount++
		case "skip":
			skipCount++
		case "unknown":
			unknownCount++
		}
	}

	data.ProvenCheckCount = passCount
	data.FailCheckCount = failCount
	data.SkipCheckCount = skipCount
	data.UnknownCheckCount = unknownCount
	data.TotalChecks = len(pack.Verification.Checks)

	return data
}

func buildImplementData(pack schema.EvidencePack) implementData {
	return implementData{
		CommitCount:      len(pack.Implementation.Commits),
		ChangedFileCount: len(pack.Implementation.ChangedFiles),
		Additions:        pack.Implementation.Additions,
		Deletions:        pack.Implementation.Deletions,
		DiffStat:         pack.Implementation.DiffStat,
		Commits:          pack.Implementation.Commits,
		ChangedFiles:     pack.Implementation.ChangedFiles,
	}
}

func buildAgentRunData(pack schema.EvidencePack) agentRunData {
	run := pack.AgentRun
	if run == nil {
		return agentRunData{Hides: true}
	}

	var inputTokens, outputTokens int64
	if run.Tokens != nil {
		inputTokens = run.Tokens.Input
		outputTokens = run.Tokens.Output
	}

	return agentRunData{
		Hides:         false,
		Provider:      run.Provider,
		Model:         run.Model,
		SessionID:     run.SessionID,
		Cost:          run.Cost,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		ToolCallCount: run.ToolCallCount,
		StartedAt:     run.StartedAt,
		EndedAt:       run.EndedAt,
		ExitStatus:    run.ExitStatus,
	}
}

func buildReportProvenanceData(pack schema.EvidencePack) reportProvenanceData {
	return reportProvenanceData{
		ShipProofVersion: pack.Provenance.ShipProofVersion,
	}
}

// buildUnexplainedData reads the pack section. An absent section renders
// nothing, because ShipProof made no measurement.
func buildUnexplainedData(pack schema.EvidencePack) unexplainedData {
	if pack.UnexplainedChange == nil {
		return unexplainedData{}
	}
	return unexplainedData{
		Present:             true,
		CoverageAvailable:   pack.UnexplainedChange.CoverageAvailable,
		LineFindings:        pack.UnexplainedChange.LineFindings,
		FileFindings:        pack.UnexplainedChange.FileFindings,
		UninstrumentedLines: pack.UnexplainedChange.UninstrumentedLines,
	}
}
