package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/schema"
)

func TestChangeReportRendersIntent(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T1")
	defer os.RemoveAll(root)

	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "SP-T1") {
		t.Error("output should contain change ID")
	}
	if !strings.Contains(html, "abc123def") {
		t.Error("output should contain snapshot hash")
	}
	if !strings.Contains(html, "observed") {
		t.Error("output should contain provenance badge")
	}
	if !strings.Contains(html, "Requirements") {
		t.Error("output should contain requirements section")
	}
}

func TestChangeReportRendersChangedSurface(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T2")
	defer os.RemoveAll(root)

	ev.Implementation = schema.ImplementationEvidence{
		Commits: []schema.ImplementationCommit{
			{Hash: "a1b2c3d4e5f67890", Author: "Alice", Subject: "Add feature X"},
		},
		ChangedFiles: []string{"src/main.go", "src/foo.go"},
		Additions:    150,
		Deletions:    42,
		DiffStat:     "2 files changed, 150 insertions(+), 42 deletions(-)",
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "Add feature X") {
		t.Error("output should contain commit subject")
	}
	if !strings.Contains(html, "src/main.go") {
		t.Error("output should contain changed file path")
	}
	if !strings.Contains(html, "+150") {
		t.Error("output should contain addition count")
	}
	if !strings.Contains(html, "-42") {
		t.Error("output should contain deletion count")
	}
}

func TestChangeReportRendersVerification(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T3")
	defer os.RemoveAll(root)

	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
			{ID: "check:lint", Status: "fail", Source: "lint", Provenance: schema.ProvenanceObserved},
			{ID: "check:audit", Status: "skip", Source: "audit", Provenance: schema.ProvenanceInferred},
			{ID: "check:human", Status: "unknown", Source: "manual", Provenance: schema.ProvenanceHuman},
		},
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "check:unit") {
		t.Error("output should contain check ID")
	}
	if !strings.Contains(html, "check:lint") {
		t.Error("output should contain failing check")
	}
	if !strings.Contains(html, "PASS") {
		t.Error("output should contain PASS label")
	}
	if !strings.Contains(html, "FAIL") {
		t.Error("output should contain FAIL label")
	}
	if !strings.Contains(html, "SKIP") {
		t.Error("output should contain SKIP label")
	}
	if !strings.Contains(html, "UNKNOWN") {
		t.Error("output should contain UNKNOWN label")
	}
	if !strings.Contains(html, "observed") {
		t.Error("output should contain observed badge")
	}
	if !strings.Contains(html, "inferred") {
		t.Error("output should contain inferred badge")
	}
	if !strings.Contains(html, "human") {
		t.Error("output should contain human badge")
	}
}

func TestChangeReportRendersAgentRun(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T4")
	defer os.RemoveAll(root)

	ev.AgentRun = &schema.AgentRunMetadata{
		Provider:      "claude",
		Model:         "claude-sonnet-4",
		SessionID:     "sess-123",
		Cost:          0.42,
		Tokens:        &schema.TokenUsageMeta{Input: 5000, Output: 1500},
		ToolCallCount: 12,
		StartedAt:     "2026-08-14T20:00:00Z",
		EndedAt:       "2026-08-14T20:05:00Z",
		ExitStatus:    "success",
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T4"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "claude-sonnet-4") {
		t.Error("output should contain model name")
	}
	if !strings.Contains(html, "$0.42") {
		t.Error("output should contain formatted cost")
	}
	if !strings.Contains(html, "sess-123") {
		t.Error("output should contain session ID")
	}
	if !strings.Contains(html, "success") {
		t.Error("output should contain exit status")
	}
}

func TestChangeReportWithoutAgentRun(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T5")
	defer os.RemoveAll(root)

	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T5"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if strings.Contains(html, "Agent Run") {
		t.Error("output should not contain Agent Run section when no agent run data exists")
	}
}

func TestChangeReportRendersUnexplainedChange(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T7")
	defer os.RemoveAll(root)

	ev.UnexplainedChange = &schema.UnexplainedEvidence{
		CoverageAvailable:   true,
		UninstrumentedLines: 61,
		LineFindings: []schema.UnexplainedLine{
			{File: "internal/git/collector.go", Symbol: "withRetry", StartLine: 190, EndLine: 207},
		},
		FileFindings: []schema.UnexplainedFile{{Path: "docs/workflow.md", IgnorePattern: "docs/**"}},
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T7"); err != nil {
		t.Fatalf("GenerateChangeReport: %v", err)
	}
	body := sb.String()
	for _, want := range []string{
		"Unexplained change",
		"withRetry",
		"190",
		"207",
		"docs/workflow.md",
		"docs/**",
		"61",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("report holds no %q", want)
		}
	}
}

func TestChangeReportMissingEvidencePack(t *testing.T) {
	root, _ := setupReportTest(t, "SP-T6")
	defer os.RemoveAll(root)

	var sb strings.Builder
	err := GenerateChangeReport(&sb, root, "SP-T6")
	if err == nil {
		t.Fatal("expected error for missing evidence pack")
	}
}

func TestPRSummaryRendersWhatChanged(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T7")
	defer os.RemoveAll(root)

	ev.Implementation = schema.ImplementationEvidence{
		Commits: []schema.ImplementationCommit{
			{Hash: "a1b2c3d4e5f67890", Author: "Alice", Subject: "Fix the widget"},
		},
		ChangedFiles: []string{"src/widget.go"},
		Additions:    10,
		Deletions:    5,
	}
	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
		},
	}
	setupEvidencePack(t, root, ev)
	setupReviewPacketPass(t, root, "SP-T7")

	var sb strings.Builder
	if err := GeneratePRSummary(&sb, root, "SP-T7"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md := sb.String()
	if !strings.Contains(md, "## What changed") {
		t.Error("output should contain What changed section")
	}
	if !strings.Contains(md, "Fix the widget") {
		t.Error("output should contain commit subject")
	}
	if !strings.Contains(md, "src/widget.go") {
		t.Error("output should contain changed file")
	}
	if !strings.Contains(md, "10") {
		t.Error("output should contain addition count")
	}
}

func TestPRSummaryRendersDeterministicEvidence(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T8")
	defer os.RemoveAll(root)

	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
		},
	}
	setupEvidencePack(t, root, ev)
	setupReviewPacketPass(t, root, "SP-T8")

	var sb strings.Builder
	if err := GeneratePRSummary(&sb, root, "SP-T8"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md := sb.String()
	if !strings.Contains(md, "## Deterministic evidence") {
		t.Error("output should contain Deterministic evidence section")
	}
	if !strings.Contains(md, "check:unit") {
		t.Error("output should contain check ID in deterministic section")
	}
	if !strings.Contains(md, "[observed]") {
		t.Error("output should contain provenance tag")
	}
}

func TestPRSummaryRendersUncertain(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T9")
	defer os.RemoveAll(root)

	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:audit", Status: "unknown", Source: "audit", Provenance: schema.ProvenanceObserved},
			{ID: "check:perf", Status: "skip", Source: "perf", Provenance: schema.ProvenanceHuman},
		},
	}
	setupEvidencePack(t, root, ev)
	setupReviewPacketPass(t, root, "SP-T9")

	var sb strings.Builder
	if err := GeneratePRSummary(&sb, root, "SP-T9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md := sb.String()
	if !strings.Contains(md, "## What remains uncertain") {
		t.Error("output should contain What remains uncertain section")
	}
	if !strings.Contains(md, "check:audit") {
		t.Error("output should contain unknown check ID")
	}
	if !strings.Contains(md, "check:perf") {
		t.Error("output should contain skip check ID")
	}
}

func TestPRSummaryRendersInspect(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T10")
	defer os.RemoveAll(root)

	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:manual", Status: "fail", Source: "manual", Provenance: schema.ProvenanceHuman},
		},
	}
	setupEvidencePack(t, root, ev)
	setupReviewPacketFail(t, root, "SP-T10")

	var sb strings.Builder
	if err := GeneratePRSummary(&sb, root, "SP-T10"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md := sb.String()
	if !strings.Contains(md, "## What to inspect") {
		t.Error("output should contain What to inspect section")
	}
	if !strings.Contains(md, "## Why each inspection matters") {
		t.Error("output should contain Why each inspection matters section")
	}
	if !strings.Contains(md, "check:manual") {
		t.Error("output should contain attention check ID in inspect section")
	}
}

func TestHTMLWellFormed(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T11")
	defer os.RemoveAll(root)

	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T11"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.HasPrefix(strings.TrimSpace(html), "<!DOCTYPE html>") {
		t.Error("HTML should start with DOCTYPE")
	}
	if !strings.Contains(html, "<html") {
		t.Error("HTML should have <html> tag")
	}
	if !strings.Contains(html, "</html>") {
		t.Error("HTML should have closing </html> tag")
	}
	if !strings.Contains(html, "<head>") && !strings.Contains(html, "</head>") {
		t.Error("HTML should have head section")
	}
	if !strings.Contains(html, "<body>") && !strings.Contains(html, "</body>") {
		t.Error("HTML should have body section")
	}
}

func TestProvenanceBadges(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T12")
	defer os.RemoveAll(root)

	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "c1", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
			{ID: "c2", Status: "pass", Source: "calc", Provenance: schema.ProvenanceDerived},
			{ID: "c3", Status: "fail", Source: "ai", Provenance: schema.ProvenanceInferred},
			{ID: "c4", Status: "skip", Source: "human", Provenance: schema.ProvenanceHuman},
		},
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GenerateChangeReport(&sb, root, "SP-T12"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := sb.String()
	if !strings.Contains(html, "prov-observed") {
		t.Error("HTML should contain prov-observed class")
	}
	if !strings.Contains(html, "prov-derived") {
		t.Error("HTML should contain prov-derived class")
	}
	if !strings.Contains(html, "prov-inferred") {
		t.Error("HTML should contain prov-inferred class")
	}
	if !strings.Contains(html, "prov-human") {
		t.Error("HTML should contain prov-human class")
	}
}

func TestPRSummaryMissingEvidencePack(t *testing.T) {
	root, _ := setupReportTest(t, "SP-T13")
	defer os.RemoveAll(root)

	var sb strings.Builder
	err := GeneratePRSummary(&sb, root, "SP-T13")
	if err == nil {
		t.Fatal("expected error for missing evidence pack")
	}
}

func TestPRSummaryReportsEmptyWhenNoChecks(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T14")
	defer os.RemoveAll(root)

	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{},
	}
	setupEvidencePack(t, root, ev)

	var sb strings.Builder
	if err := GeneratePRSummary(&sb, root, "SP-T14"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md := sb.String()
	if !strings.Contains(md, "No deterministically proven checks.") {
		t.Error("PR summary should report no deterministic checks when none exist")
	}
	if !strings.Contains(md, "No uncertain checks.") {
		t.Error("PR summary should report no uncertain checks when none exist")
	}
}

func TestPRSummaryEmptyInspectReasons(t *testing.T) {
	root, ev := setupReportTest(t, "SP-T15")
	defer os.RemoveAll(root)

	ev.Verification = schema.VerificationEvidence{
		Checks: []schema.Check{
			{ID: "check:unit", Status: "pass", Source: "junit", Provenance: schema.ProvenanceObserved},
		},
	}
	setupEvidencePack(t, root, ev)
	setupReviewPacketPass(t, root, "SP-T15")

	var sb strings.Builder
	if err := GeneratePRSummary(&sb, root, "SP-T15"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md := sb.String()
	if !strings.Contains(md, "No items require human inspection.") {
		t.Error("PR summary should say no items require inspection when none exist")
	}
}

func setupReportTest(t *testing.T, changeID string) (string, schema.EvidencePack) {
	t.Helper()

	root := t.TempDir()

	shipproofDir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(shipproofDir, 0o755); err != nil {
		t.Fatalf("create .shipproof: %v", err)
	}

	changeDir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}

	record := change.Record{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
		SourcePath:    "docs/changes/test.md",
		SnapshotPath:  ".shipproof/changes/" + changeID + "/snapshot.md",
		SHA256:        "abc123def",
		CapturedAt:    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}

	ev := schema.EvidencePack{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
		Intent: schema.IntentEvidence{
			SnapshotHash: "abc123def",
			Requirements: []schema.Requirement{
				{ID: "R1", VerificationRefs: []string{"go test -run TestR1"}},
				{ID: "R2", VerificationRefs: []string{"go test -run TestR2"}},
			},
		},
		Implementation: schema.ImplementationEvidence{
			Commits: []schema.ImplementationCommit{
				{Hash: "a1b2c3d4e5f67890abcdef1234567890abcdef", Author: "Dev", Subject: "Initial commit"},
			},
			ChangedFiles: []string{"src/main.go"},
			Additions:    100,
			Deletions:    0,
			DiffStat:     "1 file changed, 100 insertions(+)",
		},
		Verification: schema.VerificationEvidence{
			Checks: []schema.Check{},
		},
		Provenance: schema.PackProvenance{
			GeneratedAt:      "2026-08-14T20:00:00Z",
			ShipProofVersion: "0.1",
		},
	}

	return root, ev
}

func setupEvidencePack(t *testing.T, root string, ev schema.EvidencePack) {
	t.Helper()

	changeDir := filepath.Join(root, ".shipproof", "changes", ev.ChangeID)
	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		t.Fatalf("encode evidence pack: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "evidence-pack.json"), data, 0o644); err != nil {
		t.Fatalf("write evidence pack: %v", err)
	}
}

func setupReviewPacketPass(t *testing.T, root, changeID string) {
	t.Helper()

	packet := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      changeID,
		"intent": map[string]interface{}{
			"snapshot_hash":     "abc123def",
			"requirement_count": 2,
		},
		"git_summary": map[string]interface{}{
			"commits":             []interface{}{},
			"changed_files_count": 1,
			"additions":           100,
			"deletions":           0,
		},
		"already_proven": []map[string]string{
			{"id": "check:unit", "status": "pass", "source": "junit", "provenance": "observed"},
		},
		"human_attention": []interface{}{},
		"unknown":         []interface{}{},
	}

	changeDir := filepath.Join(root, ".shipproof", "changes", changeID)
	data, _ := json.MarshalIndent(packet, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(changeDir, "review-packet.json"), data, 0o644)
}

func setupReviewPacketFail(t *testing.T, root, changeID string) {
	t.Helper()

	packet := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      changeID,
		"intent": map[string]interface{}{
			"snapshot_hash":     "abc123def",
			"requirement_count": 1,
		},
		"git_summary": map[string]interface{}{
			"commits":             []interface{}{},
			"changed_files_count": 0,
			"additions":           0,
			"deletions":           0,
		},
		"already_proven": []interface{}{},
		"human_attention": []map[string]interface{}{
			{
				"check_id":              "check:manual",
				"status":                "fail",
				"provenance":            "human",
				"source":                "manual",
				"reason":                "Manual review required for this check",
				"relevant_requirements": []string{"R1"},
			},
		},
		"unknown": []map[string]string{
			{"check_id": "check:audit", "status": "unknown", "provenance": "observed", "source": "audit", "what_is_uncertain": "Outcome was not determined"},
		},
	}

	changeDir := filepath.Join(root, ".shipproof", "changes", changeID)
	data, _ := json.MarshalIndent(packet, "", "  ")
	data = append(data, '\n')
	os.WriteFile(filepath.Join(changeDir, "review-packet.json"), data, 0o644)
}
