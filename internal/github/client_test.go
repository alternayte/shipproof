package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong content-type")
		}

		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Query == "" {
			t.Error("empty query")
		}

		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{"test": true}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-token", server.URL)
	resp, err := client.execute("query { test }", nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected data")
	}
}

func TestClientMissingToken(t *testing.T) {
	t.Parallel()

	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("unexpected error message: %v", err)
	}

	_, err = NewClient("  ")
	if err == nil {
		t.Fatal("expected error for whitespace token")
	}
}

func TestClientAuthError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Bad credentials"}`))
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-token", server.URL)
	_, err := client.execute("query { test }", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
}

func TestFindPRByCommit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{
				"repository": {
					"object": {
						"associatedPullRequests": {
							"nodes": [
								{
									"number": 42,
									"url": "https://github.com/acme/widget/pull/42",
									"createdAt": "2026-08-14T10:00:00Z",
									"state": "MERGED",
									"reviews": {
										"totalCount": 2,
										"nodes": [
											{"submittedAt": "2026-08-14T12:00:00Z", "state": "APPROVED", "author": {"login": "alice"}},
											{"submittedAt": "2026-08-14T14:00:00Z", "state": "COMMENTED", "author": {"login": "bob"}}
										]
									},
									"reviewThreads": {"totalCount": 3}
								}
							]
						}
					}
				}
			}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-token", server.URL)
	pr, err := client.FindPRByCommit("acme", "widget", "abc123")
	if err != nil {
		t.Fatalf("FindPRByCommit: %v", err)
	}

	if pr.Number != 42 {
		t.Errorf("number = %d, want 42", pr.Number)
	}
	if pr.State != "MERGED" {
		t.Errorf("state = %q, want MERGED", pr.State)
	}
	if pr.Reviews.TotalCount != 2 {
		t.Errorf("reviews = %d, want 2", pr.Reviews.TotalCount)
	}
	if len(pr.Reviews.Nodes) != 2 {
		t.Fatalf("review nodes = %d, want 2", len(pr.Reviews.Nodes))
	}
	if pr.Reviews.Nodes[0].SubmittedAt != "2026-08-14T12:00:00Z" {
		t.Errorf("first review submittedAt = %q, want earliest", pr.Reviews.Nodes[0].SubmittedAt)
	}
	if pr.ReviewThreads.TotalCount != 3 {
		t.Errorf("review threads = %d, want 3", pr.ReviewThreads.TotalCount)
	}
}

func TestFindPRByCommitNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{
				"repository": {
					"object": {
						"associatedPullRequests": {"nodes": []}
					}
				}
			}`),
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-token", server.URL)
	_, err := client.FindPRByCommit("acme", "widget", "abc123")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFindPRByCommitGraphQLError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Errors: []graphQLError{{Message: "Could not resolve to a commit", Type: "NOT_FOUND"}},
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-token", server.URL)
	_, err := client.FindPRByCommit("acme", "widget", "abc123")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
