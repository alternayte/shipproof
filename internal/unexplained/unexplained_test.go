package unexplained

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/alternayte/shipproof/internal/covprofile"
	"github.com/alternayte/shipproof/internal/git"
)

func fixtureFiles() []git.FileHunks {
	return []git.FileHunks{
		{Path: "internal/git/collector.go", Hunks: []git.Hunk{
			{StartLine: 190, LineCount: 3, Symbol: "withRetry"},
		}},
		{Path: "internal/git/errors.go", Hunks: []git.Hunk{
			{StartLine: 44, LineCount: 2, Symbol: "RetryError.Error"},
		}},
		{Path: "internal/cli/app.go", Hunks: []git.Hunk{
			{StartLine: 10, LineCount: 2, Symbol: ""},
		}},
		{Path: "docs/workflow.md", Hunks: []git.Hunk{
			{StartLine: 1, LineCount: 2, Symbol: ""},
		}},
	}
}

func fixtureProfile(t *testing.T) *covprofile.Profile {
	t.Helper()
	body := "mode: set\n" +
		"m/internal/git/collector.go:190.1,192.2 3 0\n" +
		"m/internal/git/errors.go:44.1,45.2 2 0\n" +
		"m/internal/git/errors.go:60.1,61.2 1 1\n"
	profile, err := covprofile.Parse(strings.NewReader(body), "m")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return profile
}

func TestBuildAtLineLevel(t *testing.T) {
	report := Build(Input{
		ChangeID: "SP-028",
		Files:    fixtureFiles(),
		Profile:  fixtureProfile(t),
		Targets:  []string{"internal/git"},
		Ignore:   []string{"docs/**"},
	})

	if !report.CoverageAvailable {
		t.Error("CoverageAvailable = false, want true")
	}
	if len(report.LineFindings) != 2 {
		t.Fatalf("line findings = %#v", report.LineFindings)
	}
	if report.LineFindings[0] != (LineFinding{File: "internal/git/collector.go", Symbol: "withRetry", StartLine: 190, EndLine: 192}) {
		t.Errorf("first finding = %#v", report.LineFindings[0])
	}
	if report.LineFindings[1] != (LineFinding{File: "internal/git/errors.go", Symbol: "RetryError.Error", StartLine: 44, EndLine: 45}) {
		t.Errorf("second finding = %#v", report.LineFindings[1])
	}
	// app.go carries two changed lines that no block holds. workflow.md is
	// ignored, so its two lines count toward no total.
	if report.UninstrumentedLines != 2 {
		t.Errorf("uninstrumented = %d, want 2", report.UninstrumentedLines)
	}
	if len(report.FileFindings) != 2 {
		t.Fatalf("file findings = %#v", report.FileFindings)
	}
	if report.FileFindings[0] != (FileFinding{Path: "internal/cli/app.go"}) {
		t.Errorf("first file finding = %#v", report.FileFindings[0])
	}
	if report.FileFindings[1] != (FileFinding{Path: "docs/workflow.md", IgnorePattern: "docs/**"}) {
		t.Errorf("second file finding = %#v", report.FileFindings[1])
	}
}

func TestBuildDegradesToFileLevel(t *testing.T) {
	report := Build(Input{
		ChangeID: "SP-028",
		Files:    fixtureFiles(),
		Profile:  nil,
		Targets:  []string{"internal/git"},
		Ignore:   []string{"docs/**"},
	})

	if report.CoverageAvailable {
		t.Error("CoverageAvailable = true, want false")
	}
	if len(report.LineFindings) != 0 {
		t.Errorf("line findings = %#v, want none", report.LineFindings)
	}
	if report.UninstrumentedLines != 0 {
		t.Errorf("uninstrumented = %d, want 0", report.UninstrumentedLines)
	}
	if len(report.FileFindings) != 2 {
		t.Fatalf("file findings = %#v", report.FileFindings)
	}
}

func TestBuildSplitsAContiguousRun(t *testing.T) {
	body := "mode: set\n" +
		"m/a.go:10.1,10.9 1 0\n" +
		"m/a.go:11.1,11.9 1 1\n" +
		"m/a.go:12.1,12.9 1 0\n"
	profile, err := covprofile.Parse(strings.NewReader(body), "m")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := Build(Input{
		ChangeID: "SP-028",
		Files:    []git.FileHunks{{Path: "a.go", Hunks: []git.Hunk{{StartLine: 10, LineCount: 3, Symbol: "F"}}}},
		Profile:  profile,
		Targets:  []string{"a.go"},
	})
	if len(report.LineFindings) != 2 {
		t.Fatalf("findings = %#v, want two ranges", report.LineFindings)
	}
	if report.LineFindings[0].EndLine != 10 || report.LineFindings[1].StartLine != 12 {
		t.Errorf("findings = %#v", report.LineFindings)
	}
}

func TestBuildSkipsAPureDeletionHunk(t *testing.T) {
	report := Build(Input{
		ChangeID: "SP-028",
		Files:    []git.FileHunks{{Path: "a.go", Hunks: []git.Hunk{{StartLine: 10, LineCount: 0}}}},
		Profile:  fixtureProfile(t),
		Targets:  []string{"a.go"},
	})
	if len(report.LineFindings) != 0 || report.UninstrumentedLines != 0 {
		t.Errorf("report = %#v", report)
	}
}

func TestRenderLineLevelGolden(t *testing.T) {
	report := Build(Input{
		ChangeID: "SP-028",
		Files:    fixtureFiles(),
		Profile:  fixtureProfile(t),
		Targets:  []string{"internal/git"},
		Ignore:   []string{"docs/**"},
	})

	var buffer bytes.Buffer
	if err := Render(&buffer, report); err != nil {
		t.Fatalf("Render: %v", err)
	}
	want, err := os.ReadFile("testdata/line-level.txt")
	if err != nil {
		t.Fatal(err)
	}
	if buffer.String() != string(want) {
		t.Errorf("report mismatch\ngot:\n%s\nwant:\n%s", buffer.String(), want)
	}
}

func TestRenderFileLevelGolden(t *testing.T) {
	report := Build(Input{
		ChangeID: "SP-028",
		Files:    fixtureFiles(),
		Profile:  nil,
		Targets:  []string{"internal/git"},
		Ignore:   []string{"docs/**"},
	})

	var buffer bytes.Buffer
	if err := Render(&buffer, report); err != nil {
		t.Fatalf("Render: %v", err)
	}
	want, err := os.ReadFile("testdata/file-level.txt")
	if err != nil {
		t.Fatal(err)
	}
	if buffer.String() != string(want) {
		t.Errorf("report mismatch\ngot:\n%s\nwant:\n%s", buffer.String(), want)
	}
}

func TestRenderStatesTheUninstrumentedCount(t *testing.T) {
	var buffer bytes.Buffer
	if err := Render(&buffer, Report{ChangeID: "SP-028", CoverageAvailable: true, UninstrumentedLines: 61}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buffer.String(), "Not instrumented: 61 changed lines. No claim made.") {
		t.Errorf("report = %q", buffer.String())
	}
}
