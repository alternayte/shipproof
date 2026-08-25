package covprofile

import (
	"strings"
	"testing"
)

const modulePath = "github.com/alternayte/shipproof"

func TestParseFileStatesEachLine(t *testing.T) {
	profile, err := ParseFile("testdata/sample.out", modulePath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	cases := []struct {
		file string
		line int
		want State
	}{
		{"internal/git/collector.go", 29, Executed},
		{"internal/git/collector.go", 34, Executed},
		{"internal/git/collector.go", 41, NotExecuted},
		{"internal/git/collector.go", 36, NotInstrumented},
		{"internal/git/errors.go", 46, NotExecuted},
		{"internal/cli/app.go", 1, NotInstrumented},
	}
	for _, testCase := range cases {
		got := profile.Lookup(testCase.file, testCase.line)
		if got != testCase.want {
			t.Errorf("Lookup(%s, %d) = %q, want %q", testCase.file, testCase.line, got, testCase.want)
		}
	}
}

func TestExecutedBeatsNotExecutedOnOverlap(t *testing.T) {
	body := "mode: set\n" +
		"github.com/alternayte/shipproof/a.go:10.1,12.2 1 0\n" +
		"github.com/alternayte/shipproof/a.go:11.1,11.9 1 1\n"
	profile, err := Parse(strings.NewReader(body), modulePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := profile.Lookup("a.go", 11); got != Executed {
		t.Errorf("Lookup = %q, want executed", got)
	}
	if got := profile.Lookup("a.go", 12); got != NotExecuted {
		t.Errorf("Lookup = %q, want not-executed", got)
	}
}

func TestMergeSumsCounts(t *testing.T) {
	first, err := Parse(strings.NewReader("mode: set\ngithub.com/alternayte/shipproof/a.go:10.1,12.2 1 0\n"), modulePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	second, err := Parse(strings.NewReader("mode: set\ngithub.com/alternayte/shipproof/a.go:10.1,12.2 1 1\n"), modulePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	first.Merge(second)
	if got := first.Lookup("a.go", 11); got != Executed {
		t.Errorf("Lookup after merge = %q, want executed", got)
	}
}

func TestCoversReportsFilePresence(t *testing.T) {
	profile, err := ParseFile("testdata/sample.out", modulePath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if !profile.Covers("internal/git/collector.go") {
		t.Error("Covers(collector.go) = false, want true")
	}
	if profile.Covers("internal/cli/app.go") {
		t.Error("Covers(app.go) = true, want false")
	}
}

func TestParseRejectsMalformedLine(t *testing.T) {
	if _, err := Parse(strings.NewReader("mode: set\nnot a profile line\n"), modulePath); err == nil {
		t.Fatal("Parse accepted a malformed line")
	}
}

func TestMergeOfNilOtherIsSafe(t *testing.T) {
	profile, err := ParseFile("testdata/sample.out", modulePath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	profile.Merge(nil)
	if !profile.Covers("internal/git/errors.go") {
		t.Error("Covers = false after a nil merge")
	}
}
