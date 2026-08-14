package skills

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestBundledSkillsFollowOpenFormat(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "skills"))
	if err := ValidateCatalog(root); err != nil {
		t.Fatalf("ValidateCatalog: %v", err)
	}
}
