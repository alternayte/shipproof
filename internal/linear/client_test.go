package linear

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
		if r.Header.Get("Authorization") != "test-api-key" {
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

	client, _ := NewClientWithURL("test-api-key", server.URL+"/graphql")
	resp, err := client.execute("query { test }", nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected data")
	}
}

func TestClientMissingAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewClient("")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if !strings.Contains(err.Error(), "LINEAR_API_KEY") {
		t.Errorf("unexpected error message: %v", err)
	}

	_, err = NewClient("  ")
	if err == nil {
		t.Fatal("expected error for whitespace API key")
	}
}

func TestClientAuthError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, _ := NewClientWithURL("wrong-key", server.URL+"/graphql")
	_, err := client.execute("query {}", nil)
	if err == nil {
		t.Fatal("expected auth error")
	}
	var apiErr *APIError
	if err != nil && !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("unexpected error: %v", err)
	}
	_ = apiErr
}

func TestClientRateLimit(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	_, err := client.execute("query {}", nil)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClientNetworkError(t *testing.T) {
	t.Parallel()

	client, _ := NewClientWithURL("test-key", "http://invalid.test/graphql")
	_, err := client.execute("query {}", nil)
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestClientGraphQLError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`null`),
			Errors: []graphQLError{
				{Message: "Field 'x' doesn't exist on type 'Query'"},
			},
		})
	}))
	defer server.Close()

	client, _ := NewClientWithURL("test-key", server.URL+"/graphql")
	_, err := client.execute("query { x }", nil)
	if err == nil {
		t.Fatal("expected GraphQL error")
	}
}

func TestResolveAPIKeyFromEnv(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "env-key-123")
	key, err := ResolveAPIKey("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "env-key-123" {
		t.Errorf("expected env-key-123, got %s", key)
	}
}

func TestResolveAPIKeyMissing(t *testing.T) {
	_, err := ResolveAPIKey("")
	if err == nil {
		t.Fatal("expected error when LINEAR_API_KEY is missing")
	}
}
