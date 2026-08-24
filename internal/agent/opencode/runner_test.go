package opencode

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/agent"
)

var _ agent.AgentRunner = Runner{}

type stubServer struct {
	*httptest.Server
	lastPrompt   string
	lastTitle    string
	hasProviders bool
}

func newStubServer(t *testing.T, hasProviders bool) *stubServer {
	t.Helper()
	stub := &stubServer{hasProviders: hasProviders}
	mux := http.NewServeMux()
	mux.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"version": "0.5.3"})
	})
	mux.HandleFunc("/config/providers", func(w http.ResponseWriter, r *http.Request) {
		if !stub.hasProviders {
			writeJSON(w, map[string]any{"providers": []any{}})
			return
		}
		writeJSON(w, map[string]any{"providers": []map[string]string{{"id": "anthropic"}}})
	})
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		decodeBody(r, &body)
		stub.lastTitle = body["title"]
		writeJSON(w, map[string]string{"id": "ses_123"})
	})
	mux.HandleFunc("/session/ses_123/message", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Parts []struct{ Text string } `json:"parts"`
		}
		decodeBody(r, &body)
		if len(body.Parts) > 0 {
			stub.lastPrompt = body.Parts[0].Text
		}
		writeJSON(w, map[string]any{
			"info":  map[string]string{"id": "msg_1"},
			"parts": []map[string]string{{"type": "text", "text": "Implemented the change."}},
		})
	})
	stub.Server = httptest.NewServer(mux)
	t.Cleanup(stub.Close)
	return stub
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func decodeBody(r *http.Request, out any) {
	data, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(data, out)
}

func newRunner(t *testing.T, baseURL string) agent.AgentRunner {
	t.Helper()
	runner, err := New(agent.RunnerConfig{Name: Name, Settings: map[string]string{"base_url": baseURL}})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	return runner
}

// P7: the runner works over the server transport.
func TestProbeOverServerTransport(t *testing.T) {
	stub := newStubServer(t, true)
	status, err := newRunner(t, stub.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !status.Usable() {
		t.Fatalf("status = %+v", status)
	}
	if status.Version != "0.5.3" {
		t.Fatalf("version = %q", status.Version)
	}
	if status.Capabilities.ReadOnly {
		t.Fatal("OpenCode cannot enforce a read-only workspace in v0")
	}
}

func TestProbeReportsMissingProvider(t *testing.T) {
	stub := newStubServer(t, false)
	status, err := newRunner(t, stub.URL).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if !status.Installed || status.Authenticated {
		t.Fatalf("status = %+v", status)
	}
	if !strings.Contains(status.Detail, "opencode auth login") {
		t.Fatalf("detail = %q", status.Detail)
	}
}

func TestProbeReportsUnreachableServer(t *testing.T) {
	stub := newStubServer(t, true)
	url := stub.URL
	stub.Close()
	status, err := newRunner(t, url).Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if status.Installed {
		t.Fatalf("status = %+v", status)
	}
}

func TestRunCreatesSessionAndReturnsSummary(t *testing.T) {
	stub := newStubServer(t, true)
	result, err := newRunner(t, stub.URL).Run(context.Background(), agent.RunRequest{
		Workspace:    t.TempDir(),
		Change:       agent.Change{ID: "CH-020"},
		Role:         agent.RoleImplementer,
		Instructions: "Implement the approved change.",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != agent.RunStatusSuccess {
		t.Fatalf("status = %q", result.Status)
	}
	if result.SessionRef != "ses_123" {
		t.Fatalf("session ref = %q", result.SessionRef)
	}
	if result.Summary != "Implemented the change." {
		t.Fatalf("summary = %q", result.Summary)
	}
	if stub.lastTitle != "CH-020" {
		t.Fatalf("session title = %q", stub.lastTitle)
	}
	if !strings.Contains(stub.lastPrompt, "Implement the approved change.") {
		t.Fatalf("prompt = %q", stub.lastPrompt)
	}
}

func TestRunReportsFailedStatusOnServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"id": "ses_123"})
	})
	mux.HandleFunc("/session/ses_123/message", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model unavailable", http.StatusBadGateway)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	result, err := newRunner(t, server.URL).Run(context.Background(), agent.RunRequest{
		Workspace:    t.TempDir(),
		Change:       agent.Change{ID: "CH-021"},
		Role:         agent.RoleImplementer,
		Instructions: "Implement the approved change.",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Status != agent.RunStatusFailed {
		t.Fatalf("status = %q", result.Status)
	}
}
