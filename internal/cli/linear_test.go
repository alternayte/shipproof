package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestLinearIssueUsage(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear", "issue"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("expected usage message: %s", stderr.String())
	}
}

func TestLinearProjectUsage(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear", "project"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("expected usage message: %s", stderr.String())
	}
}

func TestLinearSyncUsage(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear", "sync"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("expected usage message: %s", stderr.String())
	}
}

func TestLinearUnknownSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear", "unknown"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestLinearNoSubcommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2, got %d", code)
	}
}

func TestLinearIssueMissingAPIKey(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear", "issue", "TEAM-1"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1 for missing API key, got %d", code)
	}
	if !strings.Contains(stderr.String(), "LINEAR_API_KEY") {
		t.Errorf("expected LINEAR_API_KEY error: %s", stderr.String())
	}
}

func TestLinearProjectMissingAPIKey(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear", "project", "my-project"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1 for missing API key, got %d", code)
	}
}

func TestLinearSyncMissingAPIKey(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"linear", "sync", "plan.json"}, stdout, stderr)
	if code != 1 {
		t.Errorf("expected exit 1 for missing API key, got %d", code)
	}
}
