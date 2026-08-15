package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/shipproof/shipproof/internal/schema"
)

func GenerateProjectReport(w io.Writer, root, projectName string) error {
	packs, err := scanEvidencePacks(root)
	if err != nil {
		return err
	}

	generatedAt := ""
	if len(packs) > 0 {
		generatedAt = packs[0].Provenance.GeneratedAt
	}

	data := projectReportData{
		ProjectName: projectName,
		GeneratedAt: generatedAt,
		Packs:       buildPackSummaryData(packs),
		Metrics:     buildProjectMetrics(packs),
		Unavailable: buildUnavailableMetrics(),
		Provenance:  reportProvenanceData{ShipProofVersion: schema.CurrentVersion},
	}

	return executeTemplate(w, "project_report.html", data)
}

func scanEvidencePacks(root string) ([]schema.EvidencePack, error) {
	changesDir := filepath.Join(root, ".shipproof", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no .shipproof/changes directory found; run shipproof init first")
		}
		return nil, fmt.Errorf("read changes directory: %w", err)
	}

	var packs []schema.EvidencePack

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packPath := filepath.Join(changesDir, entry.Name(), "evidence-pack.json")
		data, err := os.ReadFile(packPath)
		if err != nil {
			continue
		}

		var pack schema.EvidencePack
		if err := json.Unmarshal(data, &pack); err != nil {
			continue
		}

		packs = append(packs, pack)
	}

	if len(packs) == 0 {
		return nil, fmt.Errorf("no evidence packs found in .shipproof/changes/*/")
	}

	sort.Slice(packs, func(i, j int) bool {
		return packs[i].Provenance.GeneratedAt > packs[j].Provenance.GeneratedAt
	})

	return packs, nil
}

type projectReportData struct {
	ProjectName string
	GeneratedAt string
	Packs       []packSummaryRow
	Metrics     projectMetrics
	Unavailable []unavailableMetric
	Provenance  reportProvenanceData
}

type packSummaryRow struct {
	ChangeID     string
	GeneratedAt  string
	CheckCount   int
	PassCount    int
	FailCount    int
	HasAgentData bool
}

type projectMetrics struct {
	TotalChanges        int
	ChangesWithAgent    int
	ChangesWithoutAgent int
	TotalChecks         int
	TotalPass           int
	TotalFail           int
	TotalSkip           int
	TotalUnknown        int
	PassRate            float64
	FirstPassCount      int
	FirstPassRate       float64
	TotalRequirements   int
	CoveredRequirements int
	CoverageRate        float64
	TotalInputTokens    int64
	TotalOutputTokens   int64
	TotalToolCalls      int64
	TotalCost           float64
	Models              []string
}

type unavailableMetric struct {
	Label  string
	Reason string
}

func buildPackSummaryData(packs []schema.EvidencePack) []packSummaryRow {
	var rows []packSummaryRow
	for _, pack := range packs {
		pass := 0
		fail := 0
		for _, check := range pack.Verification.Checks {
			if check.Status == "pass" {
				pass++
			} else if check.Status == "fail" {
				fail++
			}
		}

		rows = append(rows, packSummaryRow{
			ChangeID:     pack.ChangeID,
			GeneratedAt:  pack.Provenance.GeneratedAt,
			CheckCount:   len(pack.Verification.Checks),
			PassCount:    pass,
			FailCount:    fail,
			HasAgentData: pack.AgentRun != nil,
		})
	}
	return rows
}

func buildProjectMetrics(packs []schema.EvidencePack) projectMetrics {
	m := projectMetrics{TotalChanges: len(packs)}

	modelsSet := make(map[string]struct{})

	for _, pack := range packs {
		if pack.AgentRun != nil {
			m.ChangesWithAgent++
			if pack.AgentRun.Tokens != nil {
				m.TotalInputTokens += pack.AgentRun.Tokens.Input
				m.TotalOutputTokens += pack.AgentRun.Tokens.Output
			}
			m.TotalToolCalls += pack.AgentRun.ToolCallCount
			m.TotalCost += pack.AgentRun.Cost
			if pack.AgentRun.Model != "" {
				modelsSet[pack.AgentRun.Model] = struct{}{}
			}
		} else {
			m.ChangesWithoutAgent++
		}

		for _, check := range pack.Verification.Checks {
			m.TotalChecks++
			switch check.Status {
			case "pass":
				m.TotalPass++
			case "fail":
				m.TotalFail++
			case "skip":
				m.TotalSkip++
			case "unknown":
				m.TotalUnknown++
			}
		}

		if firstPassCheck(pack) {
			m.FirstPassCount++
		}

		m.TotalRequirements += len(pack.Intent.Requirements)
		for _, req := range pack.Intent.Requirements {
			if len(req.VerificationRefs) > 0 {
				m.CoveredRequirements++
			}
		}
	}

	if m.TotalChecks > 0 {
		m.PassRate = float64(m.TotalPass) / float64(m.TotalChecks) * 100
	}
	if m.TotalChanges > 0 {
		m.FirstPassRate = float64(m.FirstPassCount) / float64(m.TotalChanges) * 100
	}
	if m.TotalRequirements > 0 {
		m.CoverageRate = float64(m.CoveredRequirements) / float64(m.TotalRequirements) * 100
	}

	for model := range modelsSet {
		m.Models = append(m.Models, model)
	}
	sort.Strings(m.Models)

	return m
}

func firstPassCheck(pack schema.EvidencePack) bool {
	for _, check := range pack.Verification.Checks {
		if check.ID == "verification:run" && check.Status == "pass" {
			return true
		}
	}
	return false
}

func buildUnavailableMetrics() []unavailableMetric {
	return []unavailableMetric{
		{Label: "Cycle time", Reason: "No cycle time data collected"},
		{Label: "Review wait", Reason: "No review wait data collected"},
		{Label: "Rework rate", Reason: "No rework data collected"},
		{Label: "Human review effort", Reason: "No human review effort data collected"},
		{Label: "Readiness blockers", Reason: "No readiness blocker history collected"},
	}
}
