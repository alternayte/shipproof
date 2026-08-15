package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/schema"
)

func TestProjectScanPacks(t *testing.T) {
	root, _ := setupProjectTest(t)

	setupProjectEvidencePack(t, root, "SP-A", "2026-08-14T20:00:00Z", 3, 2, 0)
	setupProjectEvidencePack(t, root, "SP-B", "2026-08-14T21:00:00Z", 1, 1, 0)

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "SP-A") {
		t.Error("output should list SP-A")
	}
	if !strings.Contains(html, "SP-B") {
		t.Error("output should list SP-B")
	}
	if !strings.Contains(html, "Total Changes") {
		t.Error("output should contain Total Changes")
	}
}

func TestProjectPassRate(t *testing.T) {
	root, _ := setupProjectTest(t)

	setupProjectEvidencePack(t, root, "SP-A", "2026-08-14T20:00:00Z", 4, 3, 1)
	setupProjectEvidencePack(t, root, "SP-B", "2026-08-14T21:00:00Z", 2, 1, 0)

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Total Checks") {
		t.Error("output should contain Total Checks")
	}
	if !strings.Contains(html, "passed") {
		t.Error("output should contain passed count")
	}
}

func TestProjectFirstPass(t *testing.T) {
	root, _ := setupProjectTest(t)

	setupProjectEvidencePackWithRun(t, root, "SP-A", "pass")
	setupProjectEvidencePackWithRun(t, root, "SP-B", "fail")
	setupProjectEvidencePackWithRun(t, root, "SP-C", "pass")

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "First-Pass Success") {
		t.Error("output should contain First-Pass Success")
	}
}

func TestProjectAgentUsage(t *testing.T) {
	root, _ := setupProjectTest(t)

	setupProjectEvidencePackWithAgent(t, root, "SP-A", "claude", "claude-sonnet-4", 1000, 500, 10, 0.25)
	setupProjectEvidencePackWithAgent(t, root, "SP-B", "opencode", "deepseek-v4", 2000, 800, 20, 0.50)
	setupProjectEvidencePackNoAgent(t, root, "SP-C")

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Changes With Agent Data") {
		t.Error("output should contain Changes With Agent Data")
	}
	if !strings.Contains(html, "Total Input Tokens") {
		t.Error("output should contain Total Input Tokens")
	}
	if !strings.Contains(html, "Total Tool Calls") {
		t.Error("output should contain Total Tool Calls")
	}
	if !strings.Contains(html, "Models Used") {
		t.Error("output should contain Models Used")
	}
	if !strings.Contains(html, "claude-sonnet-4") {
		t.Error("output should list model names")
	}
}

func TestProjectCost(t *testing.T) {
	root, _ := setupProjectTest(t)

	setupProjectEvidencePackWithAgent(t, root, "SP-A", "claude", "claude-sonnet-4", 100, 50, 1, 0.15)
	setupProjectEvidencePackWithAgent(t, root, "SP-B", "claude", "claude-sonnet-4", 100, 50, 1, 0.35)

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Total Cost") {
		t.Error("output should contain Total Cost")
	}
	if !strings.Contains(html, "$0.50") {
		t.Error("output should contain aggregated cost")
	}
}

func TestProjectCoverage(t *testing.T) {
	root, _ := setupProjectTest(t)

	packA := makeEvidencePack("SP-A", "2026-08-14T20:00:00Z")
	packA.Intent.Requirements = []schema.Requirement{
		{ID: "R1", VerificationRefs: []string{"go test"}},
		{ID: "R2", VerificationRefs: []string{}},
	}
	writeEvidencePack(t, root, packA)

	packB := makeEvidencePack("SP-B", "2026-08-14T21:00:00Z")
	packB.Intent.Requirements = []schema.Requirement{
		{ID: "R1", VerificationRefs: []string{"go test"}},
	}
	writeEvidencePack(t, root, packB)

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Total Requirements") {
		t.Error("output should contain Total Requirements")
	}
}

func TestProjectNoDataMarkers(t *testing.T) {
	root, _ := setupProjectTest(t)

	setupProjectEvidencePack(t, root, "SP-A", "2026-08-14T20:00:00Z", 1, 1, 0)

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if strings.Contains(html, "Unavailable Metrics") {
		t.Error("output should not contain an Unavailable Metrics section")
	}
	if !strings.Contains(html, "No review data collected") {
		t.Error("output should show a review gap notice for a pack without review data")
	}
}

func TestProjectProvenanceBadges(t *testing.T) {
	root, _ := setupProjectTest(t)

	setupProjectEvidencePack(t, root, "SP-A", "2026-08-14T20:00:00Z", 1, 1, 0)

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "prov-derived") {
		t.Error("HTML should contain prov-derived class for derived metrics")
	}
}

func TestProjectEmptyRoot(t *testing.T) {
	root := t.TempDir()

	shipproofDir := filepath.Join(root, ".shipproof")
	os.MkdirAll(shipproofDir, 0o755)
	os.MkdirAll(filepath.Join(root, ".shipproof", "changes"), 0o755)

	var sb strings.Builder
	err := GenerateProjectReport(&sb, root, "test-project")
	if err == nil {
		t.Fatal("expected error for empty changes directory")
	}
}

func setupProjectTest(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	projectName := "test-project"

	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatalf("create .shipproof: %v", err)
	}

	changesDir := filepath.Join(root, ".shipproof", "changes")
	if err := os.MkdirAll(changesDir, 0o755); err != nil {
		t.Fatalf("create changes dir: %v", err)
	}

	return root, projectName
}

func makeEvidencePack(changeID, generatedAt string) schema.EvidencePack {
	return schema.EvidencePack{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
		Intent: schema.IntentEvidence{
			SnapshotHash: "abc123",
			Requirements: []schema.Requirement{},
		},
		Implementation: schema.ImplementationEvidence{
			ChangedFiles: []string{"src/main.go"},
			Additions:    10,
			Deletions:    0,
		},
		Verification: schema.VerificationEvidence{
			Checks: []schema.Check{},
		},
		Provenance: schema.PackProvenance{
			GeneratedAt:      generatedAt,
			ShipProofVersion: "0.1",
		},
	}
}

func setupProjectEvidencePack(t *testing.T, root, changeID, generatedAt string, total, pass, fail int) {
	t.Helper()

	pack := makeEvidencePack(changeID, generatedAt)
	var checks []schema.Check
	for i := 0; i < pass; i++ {
		checks = append(checks, schema.Check{
			ID:         "check:pass-" + changeID + "-" + string(rune('a'+i)),
			Status:     "pass",
			Source:     "junit",
			Provenance: schema.ProvenanceObserved,
		})
	}
	for i := 0; i < fail; i++ {
		checks = append(checks, schema.Check{
			ID:         "check:fail-" + changeID + "-" + string(rune('a'+i)),
			Status:     "fail",
			Source:     "lint",
			Provenance: schema.ProvenanceObserved,
		})
	}
	for i := pass + fail; i < total; i++ {
		checks = append(checks, schema.Check{
			ID:         "check:skip-" + changeID + "-" + string(rune('a'+i)),
			Status:     "skip",
			Source:     "audit",
			Provenance: schema.ProvenanceInferred,
		})
	}
	pack.Verification = schema.VerificationEvidence{Checks: checks}
	writeEvidencePack(t, root, pack)
}

func setupProjectEvidencePackWithRun(t *testing.T, root, changeID, runStatus string) {
	t.Helper()

	pack := makeEvidencePack(changeID, "2026-08-14T20:00:00Z")
	pack.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "verification:run", Status: runStatus, Source: "shipproof-runner", Provenance: schema.ProvenanceObserved},
			{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
		},
	}
	writeEvidencePack(t, root, pack)
}

func setupProjectEvidencePackWithAgent(t *testing.T, root, changeID, provider, model string, inputTokens, outputTokens, toolCalls int64, cost float64) {
	t.Helper()

	pack := makeEvidencePack(changeID, "2026-08-14T20:00:00Z")
	pack.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
		},
	}
	pack.AgentRun = &schema.AgentRunMetadata{
		Provider:      provider,
		Model:         model,
		SessionID:     "sess-" + changeID,
		Cost:          cost,
		Tokens:        &schema.TokenUsageMeta{Input: inputTokens, Output: outputTokens},
		ToolCallCount: toolCalls,
		ExitStatus:    "success",
	}
	writeEvidencePack(t, root, pack)
}

func setupProjectEvidencePackNoAgent(t *testing.T, root, changeID string) {
	t.Helper()

	pack := makeEvidencePack(changeID, "2026-08-14T20:00:00Z")
	pack.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
		},
	}
	writeEvidencePack(t, root, pack)
}

func writeEvidencePack(t *testing.T, root string, pack schema.EvidencePack) {
	t.Helper()

	changeDir := filepath.Join(root, ".shipproof", "changes", pack.ChangeID)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}

	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		t.Fatalf("encode evidence pack: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "evidence-pack.json"), data, 0o644); err != nil {
		t.Fatalf("write evidence pack: %v", err)
	}
}

func TestCycleTimeEmptyCommits(t *testing.T) {
	pack := makeEvidencePack("SP-NOCOMMIT", "2026-08-14T22:00:00Z")

	entry := cycleTimeForPack(pack)
	if entry.ChangeID != "SP-NOCOMMIT" {
		t.Fatalf("change_id = %q, want SP-NOCOMMIT", entry.ChangeID)
	}
	if entry.GapNotice == "" {
		t.Fatal("expected gap notice for empty commits")
	}
	if entry.Value != "" {
		t.Fatalf("value should be empty, got %q", entry.Value)
	}
}

func TestCycleTimeWithCommits(t *testing.T) {
	pack := makeEvidencePack("SP-CYCLE", "2026-08-14T22:00:00Z")
	pack.Implementation.Commits = []schema.ImplementationCommit{
		{Hash: "abc123", Author: "dev", Timestamp: "2026-08-14T20:00:00Z", Subject: "first"},
		{Hash: "def456", Author: "dev", Timestamp: "2026-08-14T21:00:00Z", Subject: "second"},
	}

	entry := cycleTimeForPack(pack)
	if entry.ChangeID != "SP-CYCLE" {
		t.Fatalf("change_id = %q, want SP-CYCLE", entry.ChangeID)
	}
	if entry.GapNotice != "" {
		t.Fatalf("unexpected gap notice: %s", entry.GapNotice)
	}
	if entry.Value != "2.0h" {
		t.Fatalf("value = %q, want %q", entry.Value, "2.0h")
	}
}

func TestCycleTimeLongDuration(t *testing.T) {
	pack := makeEvidencePack("SP-LONG", "2026-08-17T20:00:00Z")
	pack.Implementation.Commits = []schema.ImplementationCommit{
		{Hash: "abc123", Author: "dev", Timestamp: "2026-08-14T20:00:00Z", Subject: "start"},
	}

	entry := cycleTimeForPack(pack)
	if entry.GapNotice != "" {
		t.Fatalf("unexpected gap notice: %s", entry.GapNotice)
	}
	if entry.Value != "3.0d" {
		t.Fatalf("value = %q, want %q", entry.Value, "3.0d")
	}
}

func TestCycleTimeShortDuration(t *testing.T) {
	pack := makeEvidencePack("SP-SHORT", "2026-08-14T20:30:00Z")
	pack.Implementation.Commits = []schema.ImplementationCommit{
		{Hash: "abc123", Author: "dev", Timestamp: "2026-08-14T20:00:00Z", Subject: "quick"},
	}

	entry := cycleTimeForPack(pack)
	if entry.GapNotice != "" {
		t.Fatalf("unexpected gap notice: %s", entry.GapNotice)
	}
	if entry.Value != "30m" {
		t.Fatalf("value = %q, want %q", entry.Value, "30m")
	}
}

func makeEvidencePackWithCommits(changeID, generatedAt string, timestamps ...string) schema.EvidencePack {
	pack := makeEvidencePack(changeID, generatedAt)
	for i, ts := range timestamps {
		pack.Implementation.Commits = append(pack.Implementation.Commits, schema.ImplementationCommit{
			Hash:      "hash-" + changeID + "-" + string(rune('a'+i)),
			Author:    "dev",
			Timestamp: ts,
			Subject:   "commit",
		})
	}
	return pack
}

func TestProjectCycleTime(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-A", "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z", "2026-08-14T21:00:00Z"))
	writeEvidencePack(t, root, makeEvidencePack("SP-NOCOMMIT", "2026-08-14T22:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "2.0h") {
		t.Error("output should contain the per-change cycle time")
	}
	if !strings.Contains(html, "No commit data available") {
		t.Error("output should contain a cycle time gap notice for a pack without commits")
	}
}

func TestProjectCycleTimeAverage(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-A", "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z"))
	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-B", "2026-08-14T22:00:00Z", "2026-08-14T18:00:00Z"))
	writeEvidencePack(t, root, makeEvidencePack("SP-NOCOMMIT", "2026-08-14T22:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Avg Cycle Time") {
		t.Error("output should contain Avg Cycle Time card")
	}
	if !strings.Contains(html, "3.0h") {
		t.Error("output should contain the average of 2h and 4h cycles")
	}
}

func TestProjectRework(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-A", "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z", "2026-08-14T20:30:00Z", "2026-08-14T21:00:00Z"))
	writeEvidencePack(t, root, makeEvidencePack("SP-B", "2026-08-14T22:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Total Commits") {
		t.Error("output should contain Total Commits metric")
	}

	packs, err := scanEvidencePacks(root)
	if err != nil {
		t.Fatalf("scan packs: %v", err)
	}
	rows := buildPackSummaryData(packs)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	for _, row := range rows {
		if row.ChangeID == "SP-A" && row.Commits != 3 {
			t.Errorf("SP-A commits = %d, want 3", row.Commits)
		}
		if row.ChangeID == "SP-B" && row.Commits != 0 {
			t.Errorf("SP-B commits = %d, want 0", row.Commits)
		}
	}
}

func TestProjectReworkAverage(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-A", "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z", "2026-08-14T21:00:00Z"))
	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-B", "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z", "2026-08-14T20:30:00Z", "2026-08-14T21:00:00Z", "2026-08-14T21:30:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Avg Commits Per Change") {
		t.Error("output should contain Avg Commits Per Change card")
	}
	if !strings.Contains(html, "3.0") {
		t.Error("output should contain the average commit count")
	}
}

func TestProjectCycleReworkRender(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-A", "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	for _, label := range []string{"Avg Cycle Time", "Avg Commits Per Change"} {
		idx := strings.Index(html, label)
		if idx < 0 {
			t.Errorf("output should contain %q card", label)
			continue
		}
		card := html[idx : idx+400]
		if !strings.Contains(card, "prov-derived") {
			t.Errorf("%q card should carry a derived provenance badge", label)
		}
	}
}

func TestProjectSummaryColumns(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithCommits(
		"SP-A", "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Cycle Time") {
		t.Error("summary table should contain Cycle Time column")
	}
	if !strings.Contains(html, "Commits") {
		t.Error("summary table should contain Commits column")
	}
}

func TestProjectUnavailableRemaining(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePack("SP-A", "2026-08-14T22:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if strings.Contains(html, "Unavailable Metrics") {
		t.Error("output should not contain an Unavailable Metrics section")
	}
	for _, reason := range []string{"No cycle time data collected", "No rework data collected", "No review wait data collected"} {
		if strings.Contains(html, reason) {
			t.Errorf("output should not contain %q", reason)
		}
	}
}

func makeEvidencePackWithReadiness(changeID string, blockers int) schema.EvidencePack {
	pack := makeEvidencePack(changeID, "2026-08-14T22:00:00Z")
	pack.Readiness = &schema.ReadinessEvidence{
		ShapingRef:   "session-" + strings.ToLower(changeID),
		BlockerCount: blockers,
	}
	return pack
}

func TestProjectReadinessBlockers(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithReadiness("SP-A", 2))
	writeEvidencePack(t, root, makeEvidencePackWithReadiness("SP-B", 3))
	writeEvidencePack(t, root, makeEvidencePack("SP-C", "2026-08-14T22:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Readiness Blockers") {
		t.Error("output should contain Readiness Blockers card")
	}

	packs, err := scanEvidencePacks(root)
	if err != nil {
		t.Fatalf("scan packs: %v", err)
	}
	metrics := buildProjectMetrics(packs)
	if metrics.TotalBlockers != 5 {
		t.Errorf("total blockers = %d, want 5", metrics.TotalBlockers)
	}

	rows := buildPackSummaryData(packs)
	for _, row := range rows {
		switch row.ChangeID {
		case "SP-A":
			if row.Blockers != 2 {
				t.Errorf("SP-A blockers = %d, want 2", row.Blockers)
			}
		case "SP-B":
			if row.Blockers != 3 {
				t.Errorf("SP-B blockers = %d, want 3", row.Blockers)
			}
		case "SP-C":
			if row.Blockers != 0 {
				t.Errorf("SP-C blockers = %d, want 0 for missing readiness", row.Blockers)
			}
		}
	}
}

func TestProjectReadinessRender(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithReadiness("SP-A", 2))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	idx := strings.Index(html, "Readiness Blockers")
	if idx < 0 {
		t.Fatal("output should contain Readiness Blockers card")
	}
	card := html[idx : idx+400]
	if !strings.Contains(card, "prov-derived") {
		t.Error("Readiness Blockers card should carry a derived provenance badge")
	}
	if !strings.Contains(html, "<th>Blockers</th>") {
		t.Error("summary table should contain Blockers column")
	}
	if strings.Contains(html, "No readiness blocker history collected") {
		t.Error("output should not list readiness blockers as unavailable")
	}
}

func makeEvidencePackWithReview(changeID, firstReviewAt string, comments, reviewers int, logins []string) schema.EvidencePack {
	pack := makeEvidencePackWithCommits(
		changeID, "2026-08-14T22:00:00Z", "2026-08-14T20:00:00Z")
	pack.Review = &schema.ReviewEvidence{
		Source:            "github",
		PRNumber:          1,
		PRURL:             "https://github.com/acme/widget/pull/1",
		OpenedAt:          "2026-08-14T20:30:00Z",
		FirstReviewAt:     firstReviewAt,
		ReviewCount:       reviewers,
		CommentCount:      comments,
		DistinctReviewers: reviewers,
		ReviewerLogins:    logins,
		State:             "MERGED",
		CollectedAt:       "2026-08-14T22:00:00Z",
	}
	return pack
}

func TestProjectReviewWait(t *testing.T) {
	root, _ := setupProjectTest(t)

	pack := makeEvidencePackWithReview("SP-A", "2026-08-14T23:00:00Z", 1, 1, []string{"alice"})
	pack.AgentRun = &schema.AgentRunMetadata{EndedAt: "2026-08-14T21:00:00Z"}
	writeEvidencePack(t, root, pack)

	writeEvidencePack(t, root, makeEvidencePack("SP-NOREVIEW", "2026-08-14T22:00:00Z"))

	packNoFirst := makeEvidencePackWithReview("SP-NOFIRST", "", 0, 0, nil)
	writeEvidencePack(t, root, packNoFirst)

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "2.0h") {
		t.Error("output should contain the review wait for SP-A (21:00 to 23:00)")
	}
	if !strings.Contains(html, "No review data collected") {
		t.Error("output should contain gap notice for pack without review data")
	}
	if !strings.Contains(html, "No review submitted yet") {
		t.Error("output should contain gap notice when first_review_at is empty")
	}
}

func TestProjectReviewWaitCommitFallback(t *testing.T) {
	pack := makeEvidencePackWithCommits(
		"SP-FALLBACK", "2026-08-14T22:00:00Z", "2026-08-14T18:00:00Z", "2026-08-14T19:00:00Z")
	pack.Review = &schema.ReviewEvidence{
		Source:        "github",
		PRNumber:      2,
		PRURL:         "https://github.com/acme/widget/pull/2",
		FirstReviewAt: "2026-08-14T21:00:00Z",
		CollectedAt:   "2026-08-14T22:00:00Z",
	}

	entry := reviewWaitForPack(pack)
	if entry.GapNotice != "" {
		t.Fatalf("unexpected gap notice: %s", entry.GapNotice)
	}
	if entry.Value != "2.0h" {
		t.Fatalf("value = %q, want 2.0h (latest commit 19:00 to review 21:00)", entry.Value)
	}
}

func TestProjectReviewWaitPredatesImplementation(t *testing.T) {
	pack := makeEvidencePackWithReview("SP-PRE", "2026-08-14T19:00:00Z", 1, 1, []string{"alice"})

	entry := reviewWaitForPack(pack)
	if entry.GapNotice != "Review predates implementation end" {
		t.Fatalf("gap notice = %q, want %q", entry.GapNotice, "Review predates implementation end")
	}
}

func TestProjectReviewWaitAverage(t *testing.T) {
	root, _ := setupProjectTest(t)

	packA := makeEvidencePackWithReview("SP-A", "2026-08-14T23:00:00Z", 1, 1, []string{"alice"})
	packA.AgentRun = &schema.AgentRunMetadata{EndedAt: "2026-08-14T21:00:00Z"}
	writeEvidencePack(t, root, packA)

	packB := makeEvidencePackWithReview("SP-B", "2026-08-15T03:00:00Z", 1, 1, []string{"bob"})
	packB.AgentRun = &schema.AgentRunMetadata{EndedAt: "2026-08-14T21:00:00Z"}
	writeEvidencePack(t, root, packB)

	writeEvidencePack(t, root, makeEvidencePack("SP-NOREVIEW", "2026-08-14T22:00:00Z"))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Avg Review Wait") {
		t.Error("output should contain Avg Review Wait card")
	}
	if !strings.Contains(html, "4.0h") {
		t.Error("output should contain the average of 2h and 6h review waits")
	}
}

func TestProjectReviewEffort(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithReview("SP-A", "2026-08-14T23:00:00Z", 3, 2, []string{"alice", "bob"}))
	writeEvidencePack(t, root, makeEvidencePackWithReview("SP-B", "2026-08-15T03:00:00Z", 5, 1, []string{"bob"}))
	writeEvidencePack(t, root, makeEvidencePack("SP-NOREVIEW", "2026-08-14T22:00:00Z"))

	packs, err := scanEvidencePacks(root)
	if err != nil {
		t.Fatalf("scan packs: %v", err)
	}
	metrics := buildProjectMetrics(packs)
	if metrics.TotalReviewComments != 8 {
		t.Errorf("total review comments = %d, want 8", metrics.TotalReviewComments)
	}
	if metrics.TotalReviewers != 2 {
		t.Errorf("distinct reviewer union = %d, want 2 (alice, bob)", metrics.TotalReviewers)
	}

	rows := buildPackSummaryData(packs)
	for _, row := range rows {
		switch row.ChangeID {
		case "SP-A":
			if row.ReviewComments != 3 || row.Reviewers != 2 {
				t.Errorf("SP-A review effort = %d comments, %d reviewers; want 3, 2", row.ReviewComments, row.Reviewers)
			}
		case "SP-B":
			if row.ReviewComments != 5 || row.Reviewers != 1 {
				t.Errorf("SP-B review effort = %d comments, %d reviewers; want 5, 1", row.ReviewComments, row.Reviewers)
			}
		case "SP-NOREVIEW":
			if row.ReviewComments != 0 || row.Reviewers != 0 {
				t.Errorf("SP-NOREVIEW review effort should be zero, got %d, %d", row.ReviewComments, row.Reviewers)
			}
		}
	}
}

func TestProjectReviewRender(t *testing.T) {
	root, _ := setupProjectTest(t)

	writeEvidencePack(t, root, makeEvidencePackWithReview("SP-A", "2026-08-14T23:00:00Z", 3, 1, []string{"alice"}))

	var sb strings.Builder
	if err := GenerateProjectReport(&sb, root, "test-project"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if strings.Contains(html, "Unavailable Metrics") {
		t.Error("output must not contain an Unavailable Metrics section")
	}
	for _, label := range []string{"Avg Review Wait", "Review Comments", "Distinct Reviewers"} {
		idx := strings.Index(html, label)
		if idx < 0 {
			t.Errorf("output should contain %q card", label)
			continue
		}
		card := html[idx : idx+400]
		if !strings.Contains(card, "prov-derived") {
			t.Errorf("%q card should carry a derived provenance badge", label)
		}
	}
	for _, column := range []string{"<th>Review Wait</th>", "<th>Comments</th>", "<th>Reviewers</th>"} {
		if !strings.Contains(html, column) {
			t.Errorf("summary table should contain %s", column)
		}
	}
}
