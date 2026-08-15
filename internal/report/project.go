package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

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
	CycleTime    string
	CycleGap     string
	Commits      int
	Blockers     int
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
	AvgCycleTime        string
	CycleTimeGapCount   int
	AvgCommits          float64
	TotalCommits        int
	TotalBlockers       int
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

		cycle := cycleTimeForPack(pack)

		blockers := 0
		if pack.Readiness != nil {
			blockers = pack.Readiness.BlockerCount
		}

		rows = append(rows, packSummaryRow{
			ChangeID:     pack.ChangeID,
			GeneratedAt:  pack.Provenance.GeneratedAt,
			CheckCount:   len(pack.Verification.Checks),
			PassCount:    pass,
			FailCount:    fail,
			HasAgentData: pack.AgentRun != nil,
			CycleTime:    cycle.Value,
			CycleGap:     cycle.GapNotice,
			Commits:      len(pack.Implementation.Commits),
			Blockers:     blockers,
		})
	}
	return rows
}

func buildProjectMetrics(packs []schema.EvidencePack) projectMetrics {
	m := projectMetrics{TotalChanges: len(packs)}

	modelsSet := make(map[string]struct{})
	var cycleTotal time.Duration
	var cycleCount int

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

		commits := len(pack.Implementation.Commits)
		m.TotalCommits += commits

		blockers := 0
		if pack.Readiness != nil {
			blockers = pack.Readiness.BlockerCount
		}
		m.TotalBlockers += blockers

		if duration, gap := cycleDurationForPack(pack); gap == "" {
			cycleTotal += duration
			cycleCount++
		} else {
			m.CycleTimeGapCount++
		}
	}

	if cycleCount > 0 {
		m.AvgCycleTime = formatCycleDuration(cycleTotal / time.Duration(cycleCount))
	}
	if m.TotalChanges > 0 {
		m.AvgCommits = float64(m.TotalCommits) / float64(m.TotalChanges)
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

type changeMetricEntry struct {
	ChangeID  string
	Value     string
	GapNotice string
}

func cycleTimeForPack(pack schema.EvidencePack) changeMetricEntry {
	duration, gap := cycleDurationForPack(pack)
	if gap != "" {
		return changeMetricEntry{
			ChangeID:  pack.ChangeID,
			GapNotice: gap,
		}
	}
	return changeMetricEntry{
		ChangeID: pack.ChangeID,
		Value:    formatCycleDuration(duration),
	}
}

func cycleDurationForPack(pack schema.EvidencePack) (time.Duration, string) {
	if len(pack.Implementation.Commits) == 0 {
		return 0, "No commit data available"
	}

	oldest, err := time.Parse(time.RFC3339, pack.Implementation.Commits[0].Timestamp)
	if err != nil {
		return 0, "Cannot parse commit timestamp"
	}
	for _, c := range pack.Implementation.Commits[1:] {
		t, err := time.Parse(time.RFC3339, c.Timestamp)
		if err != nil {
			continue
		}
		if t.Before(oldest) {
			oldest = t
		}
	}

	end, err := time.Parse(time.RFC3339, pack.Provenance.GeneratedAt)
	if err != nil {
		return 0, "Cannot parse evidence pack timestamp"
	}

	return end.Sub(oldest), ""
}

func formatCycleDuration(d time.Duration) string {
	hours := d.Hours()
	if hours < 1 {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if hours < 48 {
		return fmt.Sprintf("%.1fh", hours)
	}
	return fmt.Sprintf("%.1fd", hours/24)
}

func buildUnavailableMetrics() []unavailableMetric {
	return []unavailableMetric{
		{Label: "Review wait", Reason: "No review wait data collected"},
		{Label: "Human review effort", Reason: "No human review effort data collected"},
	}
}
