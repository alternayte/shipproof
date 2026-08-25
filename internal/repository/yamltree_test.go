package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sequenceFixture = `version: 1
verification:
  command: just verify
  coverage:
    command: go test -coverpkg=./... -coverprofile={{profile}} ./{{target}}/
    format: go
  unexplained_ignore:
    - "docs/**"
    - "skills/**"
`

// TestParseTreeReadsScalarKeyAlongsideSequence confirms the case that used
// to error: a scalar key sits next to a sequence key, and parseTree must
// read past the sequence without failing.
func TestParseTreeReadsScalarKeyAlongsideSequence(t *testing.T) {
	tree, err := parseTree(strings.NewReader(sequenceFixture))
	if err != nil {
		t.Fatalf("parseTree: %v", err)
	}
	value, ok := tree.get([]string{"verification", "command"})
	if !ok {
		t.Fatal("verification.command not found")
	}
	if value != "just verify" {
		t.Fatalf("verification.command = %q, want %q", value, "just verify")
	}
}

// TestGetValueReadsNestedKey confirms a key nested two levels deep, past a
// sequence sibling, still resolves through the file-backed GetValue path.
func TestGetValueReadsNestedKey(t *testing.T) {
	root := t.TempDir()
	writeLocalConfig(t, root, sequenceFixture)

	value, scope, err := GetValue(root, "verification.coverage.command")
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	if scope != ScopeLocal {
		t.Fatalf("scope = %q, want %q", scope, ScopeLocal)
	}
	want := "go test -coverpkg=./... -coverprofile={{profile}} ./{{target}}/"
	if value != want {
		t.Fatalf("verification.coverage.command = %q, want %q", value, want)
	}
}

// TestGetValueMissingKeyReturnsErrKeyNotFound confirms the caller-facing
// error for an absent key, with a sequence present elsewhere in the file.
func TestGetValueMissingKeyReturnsErrKeyNotFound(t *testing.T) {
	root := t.TempDir()
	writeLocalConfig(t, root, sequenceFixture)

	_, _, err := GetValue(root, "verification.does-not-exist")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("err = %v, want ErrKeyNotFound", err)
	}
}

// TestSetValueOnSequenceKeyFailsLoudly proves that a write to a key that
// holds a list fails and names the file and the key. yamlTree stores a list
// as opaque raw lines, so it cannot replace one with a scalar. It reported
// success before, and the value never reached disk. A lost write is
// tolerable. A lost write that reports success is not.
func TestSetValueOnSequenceKeyFailsLoudly(t *testing.T) {
	root := t.TempDir()
	writeLocalConfig(t, root, sequenceFixture)

	_, err := SetValue(root, "verification.unexplained_ignore", "scalar-value", ScopeLocal)
	if err == nil {
		t.Fatal("SetValue reported success on a key that holds a list")
	}
	if !errors.Is(err, ErrSequenceValue) {
		t.Fatalf("err = %v, want ErrSequenceValue", err)
	}
	if !strings.Contains(err.Error(), "verification.unexplained_ignore") {
		t.Errorf("the message does not name the key: %v", err)
	}
	if !strings.Contains(err.Error(), LocalConfigPath(root)) {
		t.Errorf("the message does not name the file: %v", err)
	}

	contents, err := os.ReadFile(LocalConfigPath(root))
	if err != nil {
		t.Fatal(err)
	}
	written := string(contents)
	if strings.Contains(written, "scalar-value") {
		t.Error("the refused value reached disk")
	}
	if !strings.Contains(written, `- "docs/**"`) {
		t.Error("the refused write removed the existing list")
	}
}

func writeLocalConfig(t *testing.T, root, contents string) {
	t.Helper()
	dir := filepath.Join(root, ".shipproof")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
