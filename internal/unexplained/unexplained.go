// Package unexplained reports which changed code no approved proof ran.
//
// The package claims one direction only. It never says that code answers to no
// requirement, because nothing can measure that. It says that no proof ran a
// line, and it names the lines it could not judge.
//
// Nothing here fails a change. The report is review material.
package unexplained

import (
	"strings"

	"github.com/alternayte/shipproof/internal/covprofile"
	"github.com/alternayte/shipproof/internal/git"
)

// LineFinding names one run of changed lines that no proof ran. Provenance is
// observed, because a coverage counter reported it.
type LineFinding struct {
	File string `json:"file"`
	// Symbol is the best-effort name from the Git hunk header. It is empty
	// when Git offered none.
	Symbol    string `json:"symbol,omitempty"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// FileFinding names one changed file that no proof target names and no profile
// covers. Provenance is derived.
type FileFinding struct {
	Path string `json:"path"`
	// IgnorePattern holds the pattern that excluded this path. An ignored path
	// still appears, because a silent omission would hide the ignore list from
	// a reviewer.
	IgnorePattern string `json:"ignore_pattern,omitempty"`
}

// Report is the whole signal for one change.
type Report struct {
	ChangeID string `json:"change_id"`
	// CoverageAvailable is false when no coverage command is configured. The
	// line-level section is then absent, and every changed line counts as a
	// line ShipProof could not judge.
	CoverageAvailable   bool          `json:"coverage_available"`
	LineFindings        []LineFinding `json:"line_findings"`
	FileFindings        []FileFinding `json:"file_findings"`
	UninstrumentedLines int           `json:"uninstrumented_lines"`
}

// Input holds everything Build reads.
type Input struct {
	ChangeID string
	Files    []git.FileHunks
	// Profile is nil when no coverage command is configured.
	Profile *covprofile.Profile
	// Targets holds every proof target in the verification plan.
	Targets []string
	// Ignore holds the configured glob patterns.
	Ignore []string
}

// Build derives the report. It reads no file and runs no command.
func Build(input Input) Report {
	report := Report{
		ChangeID:          input.ChangeID,
		CoverageAvailable: input.Profile != nil,
		LineFindings:      []LineFinding{},
		FileFindings:      []FileFinding{},
	}

	for _, file := range input.Files {
		pattern := ignorePattern(file.Path, input.Ignore)
		if pattern != "" {
			report.FileFindings = append(report.FileFindings,
				FileFinding{Path: file.Path, IgnorePattern: pattern})
			continue
		}

		if input.Profile != nil {
			findings, uninstrumented := judgeLines(file, input.Profile)
			report.LineFindings = append(report.LineFindings, findings...)
			report.UninstrumentedLines += uninstrumented
		} else {
			// Section 11.6 rule 4: the report states the count of lines it
			// could not judge. Without a profile it judged no line, so every
			// changed line of a file it reads counts. An ignored path counts
			// toward no total, so this loop never reaches one.
			report.UninstrumentedLines += changedLineCount(file)
		}

		if targetNames(file.Path, input.Targets) {
			continue
		}
		if input.Profile != nil && input.Profile.Covers(file.Path) {
			continue
		}
		report.FileFindings = append(report.FileFindings, FileFinding{Path: file.Path})
	}

	return report
}

// changedLineCount sums the post-image lines of one file. A pure deletion
// carries no line, so it adds nothing.
func changedLineCount(file git.FileHunks) int {
	total := 0
	for _, hunk := range file.Hunks {
		total += hunk.LineCount
	}
	return total
}

// judgeLines walks every changed line of one file and groups the lines that no
// proof ran into contiguous runs.
func judgeLines(file git.FileHunks, profile *covprofile.Profile) ([]LineFinding, int) {
	var findings []LineFinding
	uninstrumented := 0
	for _, hunk := range file.Hunks {
		open := false
		start, end := 0, 0
		for line := hunk.StartLine; line < hunk.StartLine+hunk.LineCount; line++ {
			switch profile.Lookup(file.Path, line) {
			case covprofile.NotExecuted:
				if !open {
					open, start = true, line
				}
				end = line
			case covprofile.NotInstrumented:
				uninstrumented++
				fallthrough
			default:
				if open {
					findings = append(findings, LineFinding{
						File: file.Path, Symbol: hunk.Symbol, StartLine: start, EndLine: end})
					open = false
				}
			}
		}
		if open {
			findings = append(findings, LineFinding{
				File: file.Path, Symbol: hunk.Symbol, StartLine: start, EndLine: end})
		}
	}
	return findings, uninstrumented
}

// ignorePattern returns the first pattern that excludes this path, or an empty
// string.
func ignorePattern(target string, patterns []string) string {
	for _, pattern := range patterns {
		if Match(pattern, target) {
			return pattern
		}
	}
	return ""
}

// targetNames reports whether one proof target names one file. A target names
// a file when it is that file, or when it is a directory that holds it.
func targetNames(file string, targets []string) bool {
	for _, target := range targets {
		target = strings.TrimSuffix(strings.TrimSpace(target), "/")
		if target == "" {
			continue
		}
		if file == target || strings.HasPrefix(file, target+"/") {
			return true
		}
	}
	return false
}
