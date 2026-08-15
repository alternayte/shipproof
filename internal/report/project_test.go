package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shipproof/shipproof/internal/schema"
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
	if !strings.Contains(html, "Unavailable Metrics") {
		t.Error("output should contain Unavailable Metrics section")
	}
	if !strings.Contains(html, "Cycle time") {
		t.Error("output should list Cycle time as unavailable")
	}
	if !strings.Contains(html, "Review wait") {
		t.Error("output should list Review wait as unavailable")
	}
	if !strings.Contains(html, "No cycle time data collected") {
		t.Error("output should show reason for unavailable cycle time")
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
