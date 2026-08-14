package linear

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetProject(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{
				"project": {
					"id": "proj-1",
					"name": "ShipProof v0",
					"slugId": "shipproof-v0",
					"description": "Main project",
					"state": "started",
					"lead": { "name": "Bob" },
					"url": "https://linear.app/team/project/shipproof-v0",
					"issues": {
						"nodes": [
							{
								"id": "iss-1",
								"identifier": "TEAM-1",
								"title": "Setup repo",
								"state": { "name": "Done" },
								"url": "https://linear.app/team/issue/TEAM-1"
							}
						]
					}
				}
			}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	project, err := GetProject(client, "shipproof-v0")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if project.Name != "ShipProof v0" {
		t.Errorf("name mismatch: %s", project.Name)
	}
	if project.Slug != "shipproof-v0" {
		t.Errorf("slug mismatch: %s", project.Slug)
	}
	if project.Lead != "Bob" {
		t.Errorf("lead mismatch: %s", project.Lead)
	}
	if len(project.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(project.Issues))
	}
	if project.Issues[0].Identifier != "TEAM-1" {
		t.Errorf("issue identifier mismatch: %s", project.Issues[0].Identifier)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{"project": null}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	_, err := GetProject(client, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetProjectEmptySlug(t *testing.T) {
	t.Parallel()

	client, _ := NewClientWithURL("test-key", "http://invalid.test/graphql")
	_, err := GetProject(client, "")
	if err == nil {
		t.Fatal("expected error for empty slug")
	}
	if !strings.Contains(err.Error(), "project name or slug is required") {
		t.Errorf("unexpected error: %v", err)
	}
}
