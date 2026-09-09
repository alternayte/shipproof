package report

import (
	"strings"

	"github.com/alternayte/shipproof/internal/grade"
	"github.com/alternayte/shipproof/internal/schema"
)

type changeReportData struct {
	ChangeID     string
	GeneratedAt  string
	Verdict      verdictData
	Intent       intentData
	Requirements requirementsData
	Grades       []gradeGroup
	Verify       verifyData
	Implement    implementData
	AgentRun     agentRunData
	Provenance   reportProvenanceData
	Unexplained  unexplainedData
	Attestation  attestationData
	Origin       originData
}

// verdictData holds the three-line block of Section 5. Class names the colour
// that Section 12 assigns to the verdict word.
type verdictData struct {
	Verdict string
	Reason  string
	Next    string
	Class   string
}

type requirementsData struct {
	Rows   []schema.RequirementRow
	Reason string
}

// gradeGroup holds one group of the check list. Section 12 groups the list by
// grade, and Section 6 fixes the order.
type gradeGroup struct {
	Grade string
	Note  string
	Rows  []checkRow
}

type attestationData struct {
	Signed  bool
	Subject string
	Digest  string
	Reason  string
}

type unexplainedData struct {
	Present bool
	// Measured reports whether ShipProof reached the measurement. A false
	// value means that the counts state nothing. It never means zero.
	Measured            bool
	Reason              string
	TotalLines          int
	CoverageAvailable   bool
	LineFindings        []schema.UnexplainedLine
	FileFindings        []schema.UnexplainedFile
	UninstrumentedLines int
}

type intentData struct {
	SnapshotHash     string
	SourcePath       string
	CapturedAt       string
	Stale            bool
	RequirementCount int
	Provenance       string
}

// originData answers the question every reader of the review asked: where in
// the delivery flow was this page produced, and who produced it.
type originData struct {
	Revision string
	ByBuild  bool
	Sentence string
}

type verifyData struct {
	Checks           []checkRow
	ProvenCheckCount int
	FailCheckCount   int
	// OpenCheckCount counts every check that neither passed nor failed.
	// Section 12 gives a skipped check and an unknown check the same colour,
	// because both mean unproven.
	OpenCheckCount int
	TotalChecks    int
	Reason         string
}

type checkRow struct {
	ID         string
	Status     string
	Source     string
	Detail     string
	Grade      string
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
	Reason        string
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
		SourcePath:       pack.Intent.SourcePath,
		CapturedAt:       pack.Intent.CapturedAt,
		Stale:            pack.Intent.Stale,
		RequirementCount: len(pack.Requirements),
		Provenance:       string(schema.ProvenanceObserved),
	}
}

func buildVerifyData(pack schema.EvidencePack) verifyData {
	data := verifyData{}

	var passCount, failCount, openCount int
	for _, check := range pack.Checks {
		data.Checks = append(data.Checks, checkRow{
			ID:         check.ID,
			Status:     check.Status,
			Source:     check.Source,
			Detail:     check.Detail,
			Grade:      checkGrade(check),
			Provenance: check.Provenance,
		})
		switch check.Status {
		case "pass":
			passCount++
		case "fail":
			failCount++
		case "skip", "unknown":
			openCount++
		}
	}

	data.ProvenCheckCount = passCount
	data.FailCheckCount = failCount
	data.OpenCheckCount = openCount
	data.TotalChecks = len(pack.Checks)
	if data.TotalChecks == 0 {
		data.Reason = reasonFor(pack, "checks")
	}

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
	run := pack.Agent
	if run == nil {
		return agentRunData{Hides: true, Reason: reasonFor(pack, "agent")}
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

// buildUnexplainedData reads the pack section. An unmeasured section states
// that the count is not known. It never renders a zero.
func buildUnexplainedData(pack schema.EvidencePack) unexplainedData {
	if !pack.UnexplainedChange.Measured {
		return unexplainedData{Reason: reasonFor(pack, "unexplained_change")}
	}
	total := 0
	for _, finding := range pack.UnexplainedChange.LineFindings {
		span := finding.EndLine - finding.StartLine + 1
		if span < 1 {
			span = 1
		}
		total += span
	}
	return unexplainedData{
		Present:             true,
		Measured:            true,
		TotalLines:          total,
		CoverageAvailable:   pack.UnexplainedChange.CoverageAvailable,
		LineFindings:        pack.UnexplainedChange.LineFindings,
		FileFindings:        pack.UnexplainedChange.FileFindings,
		UninstrumentedLines: pack.UnexplainedChange.UninstrumentedLines,
	}
}

// buildVerdictData reads the block that the pack recorded. The class carries
// the one colour that Section 12 assigns to the verdict word.
func buildVerdictData(pack schema.EvidencePack) verdictData {
	class := "verdict-not-proven"
	switch pack.Verdict.Verdict {
	case "PROVEN":
		class = "verdict-proven"
	case "FAILED":
		class = "verdict-failed"
	}
	return verdictData{
		Verdict: pack.Verdict.Verdict,
		Reason:  pack.Verdict.Reason,
		Next:    pack.Verdict.Next,
		Class:   class,
	}
}

func buildRequirementsData(pack schema.EvidencePack) requirementsData {
	data := requirementsData{Rows: pack.Requirements}
	if len(data.Rows) == 0 {
		data.Reason = reasonFor(pack, "requirements")
	}
	return data
}

// buildGradeGroups groups the check list by grade, in the order of Section 6.
// An empty group is omitted, because an empty table states nothing.
func buildGradeGroups(pack schema.EvidencePack) []gradeGroup {
	notes := map[string]string{
		string(grade.Observed): "A tool ran and a machine recorded each result.",
		string(grade.Stated):   "A person accepted each result and signed for it.",
		string(grade.Claimed): "Nothing confirmed these results. " +
			"A claimed check never proves a requirement.",
	}

	groups := []gradeGroup{}
	for _, name := range grade.All {
		group := gradeGroup{Grade: string(name), Note: notes[string(name)]}
		for _, check := range pack.Checks {
			if checkGrade(check) != string(name) {
				continue
			}
			group.Rows = append(group.Rows, checkRow{
				ID:         check.ID,
				Status:     check.Status,
				Source:     check.Source,
				Detail:     check.Detail,
				Grade:      string(name),
				Provenance: check.Provenance,
			})
		}
		if len(group.Rows) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

// checkGrade reads the recorded grade. A pack written before the field existed
// carries none, and the collapse rule of Section 6 supplies it.
func checkGrade(check schema.Check) string {
	if check.Grade != "" {
		return check.Grade
	}
	return string(grade.FromProvenance(check.Provenance))
}

// buildAttestationData states the signature. An unsigned pack says so, and it
// names the reason the pack recorded.
func buildAttestationData(pack schema.EvidencePack) attestationData {
	if pack.Attestation == nil || pack.Attestation.Signature == "" {
		return attestationData{Reason: reasonFor(pack, "attestation")}
	}
	return attestationData{
		Signed:  true,
		Subject: pack.Attestation.Subject,
		Digest:  pack.Attestation.Digest,
	}
}

// buildOriginData states where the page came from. A reader who does not know
// the tool cannot tell a local run from a build-system run, and all three
// readers of the review guessed.
func buildOriginData(pack schema.EvidencePack) originData {
	revision := ""
	if pack.Attestation != nil {
		revision = pack.Attestation.Subject
	}
	if revision == "" && len(pack.Implementation.Commits) > 0 {
		revision = pack.Implementation.Commits[0].Hash
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}

	signed := pack.Attestation != nil && pack.Attestation.Signature != ""
	if signed {
		return originData{
			Revision: revision,
			ByBuild:  true,
			Sentence: "A build system produced this page and signed it. That is the record an auditor reads.",
		}
	}
	return originData{
		Revision: revision,
		Sentence: "A person produced this page on a developer machine, so nothing signed it. " +
			"A build system produces the signed version when a change reaches a pull request.",
	}
}

// reasonFor reads the stated reason for an empty section. Rule 2 of Section 7
// guarantees one, and a pack that states none says so plainly.
func reasonFor(pack schema.EvidencePack, name string) string {
	if reason := strings.TrimSpace(pack.EmptySections[name]); reason != "" {
		return reason
	}
	return "The pack states no reason."
}
