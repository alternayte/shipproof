package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackWithNoChangeExitsTwo replaces the environment-dependent version of
// this test. The old one ran `pack` with no repository root, so it read
// whatever directory the runner sat in. It passed on a laptop that holds an
// untracked .shipproof and failed in continuous integration, which does not.
func TestPackWithNoChangeExitsTwo(t *testing.T) {
	newCLITestRoot(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pack"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; %s%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "shipproof start") {
		t.Errorf("the message does not name the fix: %s", stderr.String())
	}
}

// TestPackNamesTheOpenChanges keeps the ambiguity case honest. A repository
// with several open changes must name them, and it must not read as a crash.
func TestPackNamesTheOpenChanges(t *testing.T) {
	root := newCLITestRoot(t)
	for _, id := range []string{"SP-901", "SP-902"} {
		source := writeIntent(t, root, id+".md", "# "+id+"\n\nProse.\n")
		var out, errOut bytes.Buffer
		if code := Run([]string{"start", id, "--intent", source, "--ceremony", "0"}, &out, &errOut); code != 0 {
			t.Fatalf("start %s: %s", id, errOut.String())
		}
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"pack"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; %s%s", code, stdout.String(), stderr.String())
	}
	for _, id := range []string{"SP-901", "SP-902"} {
		if !strings.Contains(stderr.String(), id) {
			t.Errorf("the message never names %s: %s", id, stderr.String())
		}
	}
}

// TestNoTestDependsOnAnUntrackedRepositoryRoot guards the defect that turned
// the build red. A test that reads the developer's own .shipproof directory
// passes on a laptop and fails in continuous integration, because .shipproof
// is not committed.
func TestNoTestDependsOnAnUntrackedRepositoryRoot(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		body := string(data)
		// Every Run call needs a root. A test that sets none reads whatever
		// directory the runner happens to sit in.
		if strings.Contains(body, "Run([]string{\"pack\"}") ||
			strings.Contains(body, "Run([]string{\"prove\"}") {
			if !strings.Contains(body, "RunOverrides") && !strings.Contains(body, "newCLITestRoot") {
				t.Errorf("%s runs a command with no repository root, so it reads the runner's own directory",
					filepath.Base(entry.Name()))
			}
		}
	}
}
