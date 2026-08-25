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

// TestSetValueOnSequenceKeyDropsTheWrittenValue documents a real limitation:
// yamlTree.set does not clear sequenceItems, so setting a key that already
// holds a sequence updates the in-memory node value, but render still
// prints the old sequence and the new scalar value never reaches disk. A
// caller that runs `shipproof config set verification.unexplained_ignore
// ...` against an existing sequence loses the write silently. This test
// pins the current behaviour; it does not endorse it.
func TestSetValueOnSequenceKeyDropsTheWrittenValue(t *testing.T) {
	root := t.TempDir()
	writeLocalConfig(t, root, sequenceFixture)

	path, err := SetValue(root, "verification.unexplained_ignore", "scalar-value", ScopeLocal)
	if err != nil {
		t.Fatalf("SetValue: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	written := string(contents)

	if strings.Contains(written, "scalar-value") {
		t.Fatal("expected limitation did not reproduce: the scalar value reached disk")
	}
	if !strings.Contains(written, `- "docs/**"`) {
		t.Fatal("expected the stale sequence to remain on disk, but it is gone")
	}

	// GetValue reads the file fresh, so it reports the sequence key as
	// absent (a sequence is not a gettable scalar), not the value the
	// caller thought it had just set.
	value, _, err := GetValue(root, "verification.unexplained_ignore")
	if err == nil {
		t.Fatalf("GetValue after SetValue = %q, want ErrKeyNotFound", value)
	}
	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("err = %v, want ErrKeyNotFound", err)
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
