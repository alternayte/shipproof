package pack

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shipproof/shipproof/internal/schema"
)

func TestAssembleLoadsIntent(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.Intent.SnapshotHash != "abc123" {
		t.Errorf("expected snapshot_hash abc123, got %s", pack.Intent.SnapshotHash)
	}
	if len(pack.Intent.Requirements) != 2 {
		t.Fatalf("expected 2 requirements (1 req + 1 invariant), got %d", len(pack.Intent.Requirements))
	}
	if pack.Intent.Requirements[0].ID != "SP-005-R1" {
		t.Errorf("expected first requirement ID SP-005-R1, got %s", pack.Intent.Requirements[0].ID)
	}
	if pack.Intent.Requirements[1].ID != "INV-BASIC" {
		t.Errorf("expected second requirement ID INV-BASIC, got %s", pack.Intent.Requirements[1].ID)
	}
}

func TestAssembleLoadsRunResult(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")
	setupRunResult(t, root, "SP-005", 0)

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, check := range pack.Verification.Checks {
		if check.ID == "verification:run" {
			found = true
			if check.Status != "pass" {
				t.Errorf("expected status pass, got %s", check.Status)
			}
			if check.Provenance != schema.ProvenanceObserved {
				t.Errorf("expected provenance observed, got %s", check.Provenance)
			}
			break
		}
	}
	if !found {
		t.Error("verification:run check not found")
	}
}

func TestAssembleLoadsRunResultFail(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")
	setupRunResult(t, root, "SP-005", 1)

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, check := range pack.Verification.Checks {
		if check.ID == "verification:run" {
			found = true
			if check.Status != "fail" {
				t.Errorf("expected status fail, got %s", check.Status)
			}
			break
		}
	}
	if !found {
		t.Error("verification:run check not found")
	}
}

func TestAssembleParsesEvidence(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	junitPath := filepath.Join(root, "test-results.xml")
	writeJUnitFixture(t, junitPath)

	pack, err := Assemble(root, "SP-005", Options{EvidenceFiles: []string{junitPath}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, check := range pack.Verification.Checks {
		if check.ID == "pkg.TestOne" {
			found = true
			if check.Status != "pass" {
				t.Errorf("expected status pass, got %s", check.Status)
			}
			if check.Source != "junit" {
				t.Errorf("expected source junit, got %s", check.Source)
			}
			break
		}
	}
	if !found {
		t.Error("junit check not found in pack")
	}
}

func TestAssembleLoadsGitEvidence(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	initGitRepo(t, root)
	base := writeAndCommit(t, root, "initial.txt", "first")
	head := writeAndCommit(t, root, "second.txt", "second")

	pack, err := Assemble(root, "SP-005", Options{BaseRev: base, HeadRev: head})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pack.Implementation.Commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(pack.Implementation.Commits))
	}
	if len(pack.Implementation.ChangedFiles) != 1 {
		t.Fatalf("expected 1 changed file, got %d", len(pack.Implementation.ChangedFiles))
	}
	if pack.Implementation.Additions < 1 {
		t.Error("expected at least 1 addition")
	}

	found := false
	for _, check := range pack.Verification.Checks {
		if check.ID == "git:collect" {
			found = true
			if check.Status != "pass" {
				t.Errorf("expected git:collect pass, got %s", check.Status)
			}
			break
		}
	}
	if !found {
		t.Error("git:collect check not found")
	}
}

func TestAssemblePopulatesPack(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")
	setupRunResult(t, root, "SP-005", 0)
	initGitRepo(t, root)
	base := writeAndCommit(t, root, "hello.txt", "hello")
	head := writeAndCommit(t, root, "world.txt", "world")

	pack, err := Assemble(root, "SP-005", Options{BaseRev: base, HeadRev: head})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.SchemaVersion != "0.1" {
		t.Errorf("expected schema_version 0.1, got %s", pack.SchemaVersion)
	}
	if pack.ChangeID != "SP-005" {
		t.Errorf("expected change_id SP-005, got %s", pack.ChangeID)
	}
	if pack.Intent.SnapshotHash == "" {
		t.Error("intent.snapshot_hash is empty")
	}
	if pack.Provenance.GeneratedAt == "" {
		t.Error("provenance.generated_at is empty")
	}
	if pack.Provenance.ShipProofVersion == "" {
		t.Error("provenance.shipproof_version is empty")
	}
	if len(pack.Verification.Checks) == 0 {
		t.Error("verification checks are empty")
	}
}

func TestAssembleValidatesPack(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := pack.Validate(); err != nil {
		t.Errorf("pack did not pass validation: %v", err)
	}
}

func TestAssembleAssignsProvenance(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")
	setupRunResult(t, root, "SP-005", 0)

	junitPath := filepath.Join(root, "test-results.xml")
	writeJUnitFixture(t, junitPath)

	pack, err := Assemble(root, "SP-005", Options{EvidenceFiles: []string{junitPath}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, check := range pack.Verification.Checks {
		switch check.Provenance {
		case schema.ProvenanceObserved, schema.ProvenanceDerived, schema.ProvenanceInferred, schema.ProvenanceHuman:
		default:
			t.Errorf("check %s has invalid provenance %q", check.ID, check.Provenance)
		}
	}
}

func TestAssembleProvenanceMetadata(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.Provenance.ShipProofVersion != schema.CurrentVersion {
		t.Errorf("expected shipproof_version %s, got %s", schema.CurrentVersion, pack.Provenance.ShipProofVersion)
	}
	if pack.Provenance.GeneratedAt == "" {
		t.Error("expected non-empty generated_at")
	}
}

func TestAssembleMissingChangeRecord(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)

	_, err := Assemble(root, "SP-005", Options{})
	if err == nil {
		t.Fatal("expected error for missing change record")
	}
}

func TestAssembleMissingVerificationPlan(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")

	_, err := Assemble(root, "SP-005", Options{})
	if err == nil {
		t.Fatal("expected error for missing verification plan")
	}
}

func TestAssembleNoEvidenceFiles(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pack.Verification.Checks) != 0 {
		t.Errorf("expected 0 checks with no evidence files, got %d", len(pack.Verification.Checks))
	}
}

func TestWritePack(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-005", "abc123")
	setupVerificationPlan(t, root, "SP-005")

	pack, err := Assemble(root, "SP-005", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := WritePack(root, pack); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	packPath := filepath.Join(root, ".shipproof", "changes", "SP-005", "evidence-pack.json")
	if _, err := os.Stat(packPath); err != nil {
		t.Fatalf("evidence-pack.json not written: %v", err)
	}

	data, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("read evidence pack: %v", err)
	}

	var written schema.EvidencePack
	if err := json.Unmarshal(data, &written); err != nil {
		t.Fatalf("parse written evidence pack: %v", err)
	}
	if written.ChangeID != "SP-005" {
		t.Errorf("expected change_id SP-005, got %s", written.ChangeID)
	}
}

func TestWritePackRejectsInvalid(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)

	pack := schema.EvidencePack{
		ChangeID: "SP-005",
	}

	if err := WritePack(root, pack); err == nil {
		t.Fatal("expected error for invalid pack")
	}
}

func setupShipProofRoot(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create .shipproof: %v", err)
	}
}

func setupChangeRecord(t *testing.T, root, changeID, sha256 string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}
	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      changeID,
		"source_path":    "docs/changes/" + changeID + "-test.md",
		"snapshot_path":  ".shipproof/changes/" + changeID + "/snapshot.md",
		"sha256":         sha256,
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}
}

func setupVerificationPlan(t *testing.T, root, changeID string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}
	plan := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      changeID,
		"requirements": []map[string]interface{}{
			{
				"id":        "SP-005-R1",
				"statement": "Load intent snapshot metadata.",
				"proof": []map[string]string{
					{"type": "unit", "target": "assembler_test.go", "command": "go test -run TestX"},
				},
			},
		},
		"invariants": []map[string]interface{}{
			{
				"id":        "INV-BASIC",
				"statement": "Pack must be valid.",
				"proof": []map[string]string{
					{"type": "unit", "target": "assembler_test.go", "command": "go test -run TestInvariant"},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(plan, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "verification.json"), data, 0o644); err != nil {
		t.Fatalf("write verification plan: %v", err)
	}
}

func setupRunResult(t *testing.T, root, changeID string, exitCode int) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "runs", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create run dir: %v", err)
	}
	result := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      changeID,
		"exit_code":      exitCode,
		"duration_ms":    500,
		"stdout_path":    ".shipproof/runs/" + changeID + "/stdout.log",
		"stderr_path":    ".shipproof/runs/" + changeID + "/stderr.log",
		"timestamp":      "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "run.json"), data, 0o644); err != nil {
		t.Fatalf("write run result: %v", err)
	}
}

func writeJUnitFixture(t *testing.T, path string) {
	t.Helper()
	type junitTestCase struct {
		XMLName   xml.Name `xml:"testcase"`
		Name      string   `xml:"name,attr"`
		ClassName string   `xml:"classname,attr"`
	}
	type junitSuite struct {
		XMLName   xml.Name        `xml:"testsuite"`
		Name      string          `xml:"name,attr"`
		Tests     int             `xml:"tests,attr"`
		Failures  int             `xml:"failures,attr"`
		Errors    int             `xml:"errors,attr"`
		Skipped   int             `xml:"skipped,attr"`
		TestCases []junitTestCase `xml:"testcase"`
	}
	suite := junitSuite{
		Name:     "pkg",
		Tests:    1,
		Failures: 0,
		Errors:   0,
		Skipped:  0,
		TestCases: []junitTestCase{
			{Name: "TestOne", ClassName: "pkg"},
		},
	}
	data, _ := xml.MarshalIndent(suite, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write junit fixture: %v", err)
	}
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()
	runGitCmd(t, root, "init", "--initial-branch=main")
	runGitCmd(t, root, "config", "user.email", "test@example.com")
	runGitCmd(t, root, "config", "user.name", "Test User")
}

func writeAndCommit(t *testing.T, root, filename, content string) string {
	t.Helper()
	path := filepath.Join(root, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGitCmd(t, root, "add", filename)
	runGitCmd(t, root, "commit", "-m", "add "+filename)
	return revParseHead(t, root)
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func TestAssembleLoadsAgentRun(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-008", "abc123")
	setupVerificationPlan(t, root, "SP-008")
	setupAgentRun(t, root, "SP-008")

	pack, err := Assemble(root, "SP-008", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.AgentRun == nil {
		t.Fatal("agent_run must not be nil when agent-run.json exists")
	}
	if pack.AgentRun.Provider != "claude" {
		t.Errorf("agent_run.provider = %q, want claude", pack.AgentRun.Provider)
	}
	if pack.AgentRun.SessionID != "test-session-123" {
		t.Errorf("agent_run.session_id = %q, want test-session-123", pack.AgentRun.SessionID)
	}
	if pack.AgentRun.Model != "claude-opus-4" {
		t.Errorf("agent_run.model = %q, want claude-opus-4", pack.AgentRun.Model)
	}
}

func TestAssembleWithoutAgentRun(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-008", "abc123")
	setupVerificationPlan(t, root, "SP-008")

	pack, err := Assemble(root, "SP-008", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.AgentRun != nil {
		t.Error("agent_run must be nil when agent-run.json does not exist")
	}
}

func setupAgentRun(t *testing.T, root, changeID string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "runs", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create runs dir: %v", err)
	}
	record := map[string]interface{}{
		"provider":      "claude",
		"agent_version": "1.0.0",
		"model":         "claude-opus-4",
		"session_id":    "test-session-123",
		"started_at":    "2026-08-14T20:00:00Z",
		"ended_at":      "2026-08-14T20:30:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "agent-run.json"), data, 0o644); err != nil {
		t.Fatalf("write agent-run.json: %v", err)
	}
}

func revParseHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestAssembleReadiness(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecordWithShapingRef(t, root, "SP-012", "abc123", "test-session")
	setupVerificationPlan(t, root, "SP-012")
	setupShapingSession(t, root, "test-session", 2)

	pack, err := Assemble(root, "SP-012", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.Readiness == nil {
		t.Fatal("readiness must not be nil when shaping ref and session exist")
	}
	if pack.Readiness.ShapingRef != "test-session" {
		t.Errorf("readiness.shaping_ref = %q, want test-session", pack.Readiness.ShapingRef)
	}
	if pack.Readiness.BlockerCount != 2 {
		t.Errorf("readiness.blocker_count = %d, want 2", pack.Readiness.BlockerCount)
	}
}

func TestAssembleReadinessWithoutRef(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-012", "abc123")
	setupVerificationPlan(t, root, "SP-012")

	pack, err := Assemble(root, "SP-012", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.Readiness != nil {
		t.Error("readiness must be nil when the change record has no shaping ref")
	}
}

func TestAssembleReadinessMissingSession(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecordWithShapingRef(t, root, "SP-012", "abc123", "missing-session")
	setupVerificationPlan(t, root, "SP-012")

	pack, err := Assemble(root, "SP-012", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.Readiness != nil {
		t.Error("readiness must be nil when the shaping session file is missing")
	}
}

func setupChangeRecordWithShapingRef(t *testing.T, root, changeID, sha256, shapingRef string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}
	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      changeID,
		"source_path":    "docs/changes/" + changeID + "-test.md",
		"snapshot_path":  ".shipproof/changes/" + changeID + "/snapshot.md",
		"sha256":         sha256,
		"shaping_ref":    shapingRef,
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}
}

func setupShapingSession(t *testing.T, root, sessionID string, blockerCount int) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "shaping")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create shaping dir: %v", err)
	}

	var blockers []map[string]string
	for i := 0; i < blockerCount; i++ {
		blockers = append(blockers, map[string]string{
			"id":      "B-" + sessionID + "-" + string(rune('a'+i)),
			"summary": "blocker " + string(rune('a'+i)),
		})
	}

	session := map[string]interface{}{
		"schema_version": "0.1",
		"session_id":     sessionID,
		"subject":        "test session",
		"document_kind":  "prd",
		"state":          "shaping",
		"decisions":      []interface{}{},
		"assumptions":    []interface{}{},
		"risks":          []interface{}{},
		"unknowns":       []interface{}{},
		"readiness": map[string]interface{}{
			"blockers":           blockers,
			"decisions_required": []interface{}{},
		},
	}
	data, _ := json.MarshalIndent(session, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, sessionID+".json"), data, 0o644); err != nil {
		t.Fatalf("write shaping session: %v", err)
	}
}

func TestAssembleReview(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-014", "abc123")
	setupVerificationPlan(t, root, "SP-014")
	setupReviewFile(t, root, "SP-014")

	pack, err := Assemble(root, "SP-014", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.Review == nil {
		t.Fatal("review must not be nil when review.json exists")
	}
	if pack.Review.Source != "github" {
		t.Errorf("review.source = %q, want github", pack.Review.Source)
	}
	if pack.Review.PRNumber != 42 {
		t.Errorf("review.pr_number = %d, want 42", pack.Review.PRNumber)
	}

	found := false
	for _, check := range pack.Verification.Checks {
		if check.ID == "github:review" {
			found = true
			if check.Status != "pass" {
				t.Errorf("github:review status = %q, want pass", check.Status)
			}
			if check.Source != "github" {
				t.Errorf("github:review source = %q, want github", check.Source)
			}
			if check.Provenance != schema.ProvenanceObserved {
				t.Errorf("github:review provenance = %q, want observed", check.Provenance)
			}
		}
	}
	if !found {
		t.Error("github:review check not found")
	}
}

func TestAssembleWithoutReview(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-014", "abc123")
	setupVerificationPlan(t, root, "SP-014")

	pack, err := Assemble(root, "SP-014", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pack.Review != nil {
		t.Error("review must be nil when review.json does not exist")
	}
	for _, check := range pack.Verification.Checks {
		if check.ID == "github:review" {
			t.Error("github:review check must not exist without review.json")
		}
	}
}

func TestAssembleMalformedReview(t *testing.T) {
	root := t.TempDir()
	setupShipProofRoot(t, root)
	setupChangeRecord(t, root, "SP-014", "abc123")
	setupVerificationPlan(t, root, "SP-014")

	dir := filepath.Join(root, ".shipproof", "changes", "SP-014")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write malformed review.json: %v", err)
	}

	_, err := Assemble(root, "SP-014", Options{})
	if err == nil {
		t.Fatal("expected error for malformed review.json")
	}
}

func setupReviewFile(t *testing.T, root, changeID string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}
	review := map[string]interface{}{
		"source":             "github",
		"pr_number":          42,
		"pr_url":             "https://github.com/acme/widget/pull/42",
		"opened_at":          "2026-08-14T10:00:00Z",
		"first_review_at":    "2026-08-14T12:00:00Z",
		"review_count":       1,
		"comment_count":      3,
		"distinct_reviewers": 1,
		"reviewer_logins":    []string{"alice"},
		"state":              "MERGED",
		"collected_at":       "2026-08-14T13:00:00Z",
	}
	data, _ := json.MarshalIndent(review, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "review.json"), data, 0o644); err != nil {
		t.Fatalf("write review.json: %v", err)
	}
}
