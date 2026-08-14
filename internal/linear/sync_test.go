package linear

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPlanFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	planFile := filepath.Join(root, "plan.json")
	plan := []PlanIssue{
		{Title: "Backend API", Description: "Build the REST API"},
		{Title: "Frontend UI", Description: "Build the React app", Children: []PlanIssue{
			{Title: "Login page"},
			{Title: "Dashboard"},
		}},
	}
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(planFile, data, 0o644)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{
				"projectCreate": {
					"project": { "id": "proj-new", "name": "Test", "url": "https://linear.app/project/1" }
				}
			}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader("y\n")

	result, err := SyncPlan(client, planFile, "team-1", stdin, stderr)
	if err != nil {
		t.Fatalf("SyncPlan: %v", err)
	}
	if result.ProjectID != "proj-new" {
		t.Errorf("project ID mismatch: %s", result.ProjectID)
	}
}

func TestLoadPlanFileMissing(t *testing.T) {
	t.Parallel()

	client, _ := NewClientWithURL("test-key", "http://invalid.test/graphql")
	stderr := &bytes.Buffer{}
	stdin := bytes.NewReader(nil)

	_, err := SyncPlan(client, "/nonexistent/plan.json", "team-1", stdin, stderr)
	if err == nil {
		t.Fatal("expected error for missing plan file")
	}
	if !strings.Contains(err.Error(), "read plan file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSyncCreatesProject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	planFile := filepath.Join(root, "plan.json")
	plan := []PlanIssue{{Title: "One issue"}}
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(planFile, data, 0o644)

	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		json.NewDecoder(r.Body).Decode(&req)
		if receivedQuery == "" {
			receivedQuery = req.Query
		}
		if strings.Contains(req.Query, "projectCreate") {
			json.NewEncoder(w).Encode(graphQLResponse{
				Data: json.RawMessage(`{
					"projectCreate": {
						"project": { "id": "proj-abc", "name": "Test", "url": "https://linear.app/project/proj-abc" }
					}
				}`),
			})
		} else if strings.Contains(req.Query, "issueCreate") {
			json.NewEncoder(w).Encode(graphQLResponse{
				Data: json.RawMessage(`{
					"issueCreate": {
						"issue": { "id": "iss-1", "identifier": "TEAM-55", "title": "One issue", "url": "https://linear.app/issue/TEAM-55" }
					}
				}`),
			})
		}
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader("yes\n")

	result, err := SyncPlan(client, planFile, "team-1", stdin, stderr)
	if err != nil {
		t.Fatalf("SyncPlan: %v", err)
	}
	if result.ProjectID != "proj-abc" {
		t.Errorf("expected project proj-abc, got %s", result.ProjectID)
	}
	if len(result.CreatedIssues) != 1 {
		t.Errorf("expected 1 created issue, got %d", len(result.CreatedIssues))
	}
}

func TestSyncCreatesIssuesWithRelationships(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	planFile := filepath.Join(root, "plan.json")
	plan := []PlanIssue{
		{
			Title: "Parent Issue",
			Children: []PlanIssue{
				{Title: "Child Issue 1"},
				{Title: "Child Issue 2"},
			},
		},
	}
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(planFile, data, 0o644)

	issueCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		json.NewDecoder(r.Body).Decode(&req)
		if strings.Contains(req.Query, "projectCreate") {
			json.NewEncoder(w).Encode(graphQLResponse{
				Data: json.RawMessage(`{
					"projectCreate": {
						"project": { "id": "proj-rel", "name": "Test", "url": "https://linear.app/project/1" }
					}
				}`),
			})
		} else {
			issueCount++
			issueID := []string{"TEAM-100", "TEAM-101", "TEAM-102"}[issueCount-1]
			resp := map[string]interface{}{
				"issueCreate": map[string]interface{}{
					"issue": map[string]string{
						"id":         "iss-" + string(rune('0'+issueCount)),
						"identifier": issueID,
						"title":      "Test",
						"url":        "https://linear.app/issue/" + issueID,
					},
				},
			}
			respData, _ := json.Marshal(resp)
			json.NewEncoder(w).Encode(graphQLResponse{Data: respData})
		}
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader("y\n")

	result, err := SyncPlan(client, planFile, "team-1", stdin, stderr)
	if err != nil {
		t.Fatalf("SyncPlan: %v", err)
	}
	if len(result.CreatedIssues) != 3 {
		t.Errorf("expected 3 created issues, got %d", len(result.CreatedIssues))
	}
}

func TestSyncConfirmReject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	planFile := filepath.Join(root, "plan.json")
	plan := []PlanIssue{{Title: "Test issue"}}
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(planFile, data, 0o644)

	client, _ := NewClientWithURL("test-key", "http://invalid.test/graphql")
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader("n\n")

	_, err := SyncPlan(client, planFile, "team-1", stdin, stderr)
	if err == nil {
		t.Fatal("expected sync cancelled error")
	}
	if !strings.Contains(err.Error(), "sync cancelled") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSyncConfirmApprove(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	planFile := filepath.Join(root, "plan.json")
	plan := []PlanIssue{{Title: "Test issue"}}
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(planFile, data, 0o644)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		json.NewDecoder(r.Body).Decode(&req)
		if strings.Contains(req.Query, "projectCreate") {
			json.NewEncoder(w).Encode(graphQLResponse{
				Data: json.RawMessage(`{
					"projectCreate": {
						"project": { "id": "proj-yes", "name": "Test", "url": "https://linear.app/p/1" }
					}
				}`),
			})
		} else {
			json.NewEncoder(w).Encode(graphQLResponse{
				Data: json.RawMessage(`{
					"issueCreate": {
						"issue": { "id": "iss-yes", "identifier": "TEAM-200", "title": "Test issue", "url": "https://linear.app/i/TEAM-200" }
					}
				}`),
			})
		}
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader("yes\n")

	result, err := SyncPlan(client, planFile, "team-1", stdin, stderr)
	if err != nil {
		t.Fatalf("SyncPlan: %v", err)
	}
	if result.ProjectID == "" {
		t.Error("expected project to be created")
	}
}
