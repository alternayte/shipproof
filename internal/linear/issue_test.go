package linear

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetIssue(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{
				"issue": {
					"id": "abc-123",
					"identifier": "TEAM-42",
					"title": "Fix login bug",
					"description": "Users cannot log in.",
					"state": { "name": "In Progress" },
					"assignee": { "name": "Alice" },
					"cycle": { "number": 3 },
					"labels": { "nodes": [{"name": "bug"}, {"name": "p1"}] },
					"url": "https://linear.app/team/issue/TEAM-42",
					"attachments": { "nodes": [] }
				}
			}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	issue, err := GetIssue(client, "TEAM-42")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Identifier != "TEAM-42" {
		t.Errorf("expected TEAM-42, got %s", issue.Identifier)
	}
	if issue.Title != "Fix login bug" {
		t.Errorf("title mismatch: %s", issue.Title)
	}
	if issue.State != "In Progress" {
		t.Errorf("state mismatch: %s", issue.State)
	}
	if issue.Assignee != "Alice" {
		t.Errorf("assignee mismatch: %s", issue.Assignee)
	}
	if issue.CycleNumber != 3 {
		t.Errorf("cycle mismatch: %d", issue.CycleNumber)
	}
	if len(issue.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issue.Labels))
	}
}

func TestGetIssueNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{"issue": null}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	_, err := GetIssue(client, "TEAM-99")
	if err == nil {
		t.Fatal("expected error for missing issue")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetIssueInvalidIdentifier(t *testing.T) {
	t.Parallel()

	client, _ := NewClientWithURL("test-key", "http://invalid.test/graphql")
	_, err := GetIssue(client, "bad")
	if err == nil {
		t.Fatal("expected error for invalid identifier")
	}
	if !strings.Contains(err.Error(), "invalid issue identifier") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetIssueWithDocuments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{
				"issue": {
					"id": "def-456",
					"identifier": "TEAM-10",
					"title": "Add feature",
					"description": "New feature",
					"state": { "name": "Todo" },
					"assignee": null,
					"cycle": null,
					"labels": { "nodes": [] },
					"url": "https://linear.app/team/issue/TEAM-10",
					"attachments": {
						"nodes": [
							{ "title": "PRD draft", "url": "https://linear.app/doc/1" },
							{ "title": "Architecture notes", "url": "https://linear.app/doc/2" }
						]
					}
				}
			}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	issue, err := GetIssue(client, "TEAM-10")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if len(issue.Documents) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(issue.Documents))
	}
	if issue.Documents[0].Title != "PRD draft" {
		t.Errorf("document title mismatch: %s", issue.Documents[0].Title)
	}
}

func TestGetIssueWithoutDocuments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{
				"issue": {
					"id": "ghi-789",
					"identifier": "TEAM-20",
					"title": "Simple task",
					"description": "",
					"state": { "name": "Todo" },
					"assignee": null,
					"cycle": null,
					"labels": { "nodes": [] },
					"url": "https://linear.app/team/issue/TEAM-20",
					"attachments": { "nodes": [] }
				}
			}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	issue, err := GetIssue(client, "TEAM-20")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Documents != nil {
		t.Errorf("expected Documents to be nil for empty attachments, got %v", issue.Documents)
	}

	body, _ := json.Marshal(issue)
	if strings.Contains(string(body), `"documents":null`) {
		t.Error("documents field should be omitted when empty, not null")
	}
}
