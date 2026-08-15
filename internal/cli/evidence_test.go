package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidencePack(t *testing.T) {
	root := t.TempDir()
	setupTestRepo(t, root)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "pack", "SP-005"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Evidence pack written:") {
		t.Errorf("expected success message, got: %s", output)
	}

	packPath := filepath.Join(root, ".shipproof", "changes", "SP-005", "evidence-pack.json")
	data, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatalf("evidence-pack.json not written: %v", err)
	}

	var pack struct {
		ChangeID string `json:"change_id"`
	}
	if err := json.Unmarshal(data, &pack); err != nil {
		t.Fatalf("parse evidence pack: %v", err)
	}
	if pack.ChangeID != "SP-005" {
		t.Errorf("expected change_id SP-005, got %s", pack.ChangeID)
	}
}

func TestEvidencePackMissingChangeID(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "pack"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestEvidencePackMissingChange(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	RunOverrides["."] = root
	defer delete(RunOverrides, ".")

	code := Run([]string{"evidence", "pack", "SP-999"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
}

func TestEvidencePackMissingVerificationPlan(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestChangeRecord(t, root, "SP-099")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	RunOverrides["."] = root
	defer delete(RunOverrides, ".")

	code := Run([]string{"evidence", "pack", "SP-099"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
}

func setupTestRepo(t *testing.T, root string) {
	t.Helper()
	setupShipProofDir(t, root)

	changeDir := filepath.Join(root, ".shipproof", "changes", "SP-005")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}

	record := map[string]string{
		"schema_version": "0.1",
		"change_id":      "SP-005",
		"source_path":    "docs/changes/SP-005-test.md",
		"snapshot_path":  ".shipproof/changes/SP-005/snapshot.md",
		"sha256":         "abc123",
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}

	plan := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      "SP-005",
		"requirements": []map[string]interface{}{
			{
				"id":        "SP-005-R1",
				"statement": "Load intent snapshot metadata.",
				"proof": []map[string]string{
					{"type": "unit", "target": "test.go", "command": "go test"},
				},
			},
		},
		"invariants": []map[string]interface{}{
			{
				"id":        "INV-TEST",
				"statement": "Must be valid.",
				"proof": []map[string]string{
					{"type": "unit", "target": "test.go", "command": "go test"},
				},
			},
		},
	}
	data, _ = json.MarshalIndent(plan, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(changeDir, "verification.json"), data, 0o644); err != nil {
		t.Fatalf("write verification plan: %v", err)
	}

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })
}

func setupShipProofDir(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("create .shipproof: %v", err)
	}
}

func writeTestChangeRecord(t *testing.T, root, changeID string) {
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
		"sha256":         "abc123",
		"captured_at":    "2026-08-14T20:00:00Z",
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "change.json"), data, 0o644); err != nil {
		t.Fatalf("write change record: %v", err)
	}
}

func writeTestEvidencePackWithCommits(t *testing.T, root, changeID string, commits []map[string]string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof", "changes", changeID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create change dir: %v", err)
	}
	pack := map[string]interface{}{
		"schema_version": "0.1",
		"change_id":      changeID,
		"intent": map[string]interface{}{
			"snapshot_hash": "abc123",
			"requirements":  []interface{}{},
		},
		"implementation": map[string]interface{}{
			"commits":       commits,
			"changed_files": []string{"main.go"},
			"additions":     1,
			"deletions":     0,
			"diff_stat":     "main.go | 1 +",
		},
		"verification": map[string]interface{}{
			"checks": []interface{}{},
		},
		"provenance": map[string]string{
			"generated_at":      "2026-08-14T20:00:00Z",
			"shipproof_version": "0.1",
		},
	}
	data, _ := json.MarshalIndent(pack, "", "  ")
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, "evidence-pack.json"), data, 0o644); err != nil {
		t.Fatalf("write evidence pack: %v", err)
	}
}

func TestEvidenceReviewMissingArgs(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "review"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestEvidenceReviewMissingPack(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "review", "SP-777"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "evidence pack") {
		t.Errorf("expected evidence pack error, got: %s", stderr.String())
	}
}

func TestEvidenceReviewNoToken(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestEvidencePackWithCommits(t, root, "SP-777", []map[string]string{
		{"hash": "abc123", "author": "dev", "timestamp": "2026-08-14T10:00:00Z", "subject": "commit"},
	})

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	t.Setenv("GITHUB_TOKEN", "")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "review", "SP-777"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "GITHUB_TOKEN") {
		t.Errorf("expected GITHUB_TOKEN error, got: %s", stderr.String())
	}
}

func TestEvidenceReviewNoCommits(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestEvidencePackWithCommits(t, root, "SP-777", []map[string]string{})

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	t.Setenv("GITHUB_TOKEN", "test-token")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "review", "SP-777"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no commits") {
		t.Errorf("expected no commits error, got: %s", stderr.String())
	}
}

func TestEvidenceReviewNoRemote(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestEvidencePackWithCommits(t, root, "SP-777", []map[string]string{
		{"hash": "abc123", "author": "dev", "timestamp": "2026-08-14T10:00:00Z", "subject": "commit"},
	})

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	t.Setenv("GITHUB_TOKEN", "test-token")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "review", "SP-777"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no origin remote") {
		t.Errorf("expected no origin remote error, got: %s", stderr.String())
	}
}

func TestEvidenceReviewWritesFile(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestEvidencePackWithCommits(t, root, "SP-777", []map[string]string{
		{"hash": "abc123", "author": "dev", "timestamp": "2026-08-14T10:00:00Z", "subject": "commit"},
	})

	gitDir := root
	runGitInit(t, gitDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"object": map[string]interface{}{
						"associatedPullRequests": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"number":    42,
									"url":       "https://github.com/acme/widget/pull/42",
									"createdAt": "2026-08-14T10:00:00Z",
									"state":     "MERGED",
									"reviews": map[string]interface{}{
										"totalCount": 1,
										"nodes": []map[string]interface{}{
											{"submittedAt": "2026-08-14T12:00:00Z", "state": "APPROVED", "author": map[string]string{"login": "alice"}},
										},
									},
									"reviewThreads": map[string]interface{}{"totalCount": 2},
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	githubAPIURLOverride = server.URL
	t.Cleanup(func() { githubAPIURLOverride = "" })

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	t.Setenv("GITHUB_TOKEN", "test-token")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "review", "SP-777"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}

	reviewPath := filepath.Join(root, ".shipproof", "changes", "SP-777", "review.json")
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("review.json not written: %v", err)
	}

	var review struct {
		Source            string `json:"source"`
		PRNumber          int    `json:"pr_number"`
		FirstReviewAt     string `json:"first_review_at"`
		ReviewCount       int    `json:"review_count"`
		CommentCount      int    `json:"comment_count"`
		DistinctReviewers int    `json:"distinct_reviewers"`
	}
	if err := json.Unmarshal(data, &review); err != nil {
		t.Fatalf("parse review.json: %v", err)
	}
	if review.Source != "github" {
		t.Errorf("source = %q, want github", review.Source)
	}
	if review.PRNumber != 42 {
		t.Errorf("pr_number = %d, want 42", review.PRNumber)
	}
	if review.FirstReviewAt != "2026-08-14T12:00:00Z" {
		t.Errorf("first_review_at = %q, want 2026-08-14T12:00:00Z", review.FirstReviewAt)
	}
	if review.ReviewCount != 1 {
		t.Errorf("review_count = %d, want 1", review.ReviewCount)
	}
	if review.CommentCount != 3 {
		t.Errorf("comment_count = %d, want 3 (1 review + 2 threads)", review.CommentCount)
	}
	if review.DistinctReviewers != 1 {
		t.Errorf("distinct_reviewers = %d, want 1", review.DistinctReviewers)
	}

	if !strings.Contains(stdout.String(), "PR: #42") {
		t.Errorf("expected PR summary in stdout, got: %s", stdout.String())
	}
}

func TestEvidenceReviewNoPRFound(t *testing.T) {
	root := t.TempDir()
	setupShipProofDir(t, root)
	writeTestEvidencePackWithCommits(t, root, "SP-777", []map[string]string{
		{"hash": "abc123", "author": "dev", "timestamp": "2026-08-14T10:00:00Z", "subject": "commit"},
	})

	runGitInit(t, root)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"object": map[string]interface{}{
						"associatedPullRequests": map[string]interface{}{"nodes": []interface{}{}},
					},
				},
			},
		})
	}))
	defer server.Close()

	githubAPIURLOverride = server.URL
	t.Cleanup(func() { githubAPIURLOverride = "" })

	RunOverrides["."] = root
	t.Cleanup(func() { delete(RunOverrides, ".") })

	t.Setenv("GITHUB_TOKEN", "test-token")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"evidence", "review", "SP-777"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no GitHub pull request found") {
		t.Errorf("expected no PR found error, got: %s", stderr.String())
	}
}

func runGitInit(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "--initial-branch=main")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/acme/widget.git")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, string(out))
	}
}
