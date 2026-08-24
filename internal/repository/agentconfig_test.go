package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newScopes(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	if err := os.MkdirAll(filepath.Join(root, ".shipproof"), 0o755); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	return root
}

// P3: configuration reads and writes at local and global scope.
func TestSetAndGetAcrossScopes(t *testing.T) {
	root := newScopes(t)

	if _, err := SetValue(root, "agent.runner", "codex", ScopeGlobal); err != nil {
		t.Fatalf("set global: %v", err)
	}
	value, scope, err := GetValue(root, "agent.runner")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if value != "codex" || scope != ScopeGlobal {
		t.Fatalf("got %q from %s, want codex from global", value, scope)
	}

	if _, err := SetValue(root, "agent.runner", "opencode", ScopeLocal); err != nil {
		t.Fatalf("set local: %v", err)
	}
	value, scope, err = GetValue(root, "agent.runner")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if value != "opencode" || scope != ScopeLocal {
		t.Fatalf("got %q from %s, want opencode from local", value, scope)
	}
}

func TestGetMissingKey(t *testing.T) {
	root := newScopes(t)
	if _, _, err := GetValue(root, "agent.runner"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("want ErrKeyNotFound, got %v", err)
	}
}

func TestSetPreservesExistingContent(t *testing.T) {
	root := newScopes(t)
	original := "# ShipProof configuration\nverification:\n  command: \"just verify\"\nevidence:\n  capture: metadata\n"
	if err := os.WriteFile(LocalConfigPath(root), []byte(original), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	if _, err := SetValue(root, "agent.repair.max_attempts", "3", ScopeLocal); err != nil {
		t.Fatalf("set: %v", err)
	}

	data, err := os.ReadFile(LocalConfigPath(root))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(data)
	for _, want := range []string{"# ShipProof configuration", "just verify", "capture: metadata", "max_attempts: 3"} {
		if !strings.Contains(text, want) {
			t.Fatalf("config lost %q:\n%s", want, text)
		}
	}

	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatalf("existing loader broke: %v", err)
	}
	if cfg.Verification.Command != "just verify" || cfg.Evidence.Capture != CaptureMetadata {
		t.Fatalf("existing loader read %+v", cfg)
	}
}

func TestLoadAgentConfigMergesScopes(t *testing.T) {
	root := newScopes(t)
	if _, err := SetValue(root, "agent.runner", "codex", ScopeGlobal); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := SetValue(root, "agent.review_runner", "claude", ScopeGlobal); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := SetValue(root, "agent.runners.codex.model", "o4-mini", ScopeLocal); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, err := SetValue(root, "agent.runner", "opencode", ScopeLocal); err != nil {
		t.Fatalf("set: %v", err)
	}

	config, err := LoadAgentConfig(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if config.Runner != "opencode" {
		t.Fatalf("runner = %q, want opencode", config.Runner)
	}
	if config.ReviewRunner != "claude" {
		t.Fatalf("review runner = %q, want claude", config.ReviewRunner)
	}
	if config.RepairMaxAttempts != DefaultRepairAttempts {
		t.Fatalf("repair attempts = %d, want %d", config.RepairMaxAttempts, DefaultRepairAttempts)
	}
	if config.Runners["codex"]["model"] != "o4-mini" {
		t.Fatalf("runner settings = %+v", config.Runners)
	}
}

func TestValidateKeyRejectsCredentials(t *testing.T) {
	for _, key := range []string{"agent.api_key", "agent.runners.codex.token", "linear.secret", "agent.password"} {
		if err := ValidateKey(key); !errors.Is(err, ErrCredentialKey) {
			t.Fatalf("key %q: want ErrCredentialKey, got %v", key, err)
		}
	}
	if err := ValidateKey("agent.runner"); err != nil {
		t.Fatalf("agent.runner must be valid: %v", err)
	}
	if err := ValidateKey("Agent.Runner"); err == nil {
		t.Fatal("uppercase key must fail")
	}
}
