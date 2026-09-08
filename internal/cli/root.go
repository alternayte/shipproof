package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunOverrides maps a request path to a repository root. A test sets it to
// point the command set at a temporary directory.
var RunOverrides = map[string]string{}

// findRepositoryRoot walks up from path and returns the first directory that
// holds a .shipproof directory.
func findRepositoryRoot(path string) (string, error) {
	if overridden, ok := RunOverrides[path]; ok {
		return overridden, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, ".shipproof")); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("ShipProof repository root not found")
		}
		abs = parent
	}
}
