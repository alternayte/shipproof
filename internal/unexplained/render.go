package unexplained

import (
	"fmt"
	"io"
)

// Render writes the report in the form Section 11.4 shows.
func Render(writer io.Writer, report Report) error {
	buffer := &errorWriter{writer: writer}

	buffer.printf("Unexplained change — %s\n", report.ChangeID)

	if report.CoverageAvailable {
		if len(report.LineFindings) > 0 {
			pathWidth, symbolWidth := lineColumnWidths(report.LineFindings)
			buffer.printf("\n  No proof ran these (observed):\n")
			for _, finding := range report.LineFindings {
				buffer.printf("    %-*s  %-*s  lines %d-%d\n",
					pathWidth, finding.File, symbolWidth, finding.Symbol,
					finding.StartLine, finding.EndLine)
			}
		}
	} else {
		buffer.printf("\n  No coverage command is configured. ShipProof cannot see inside a file.\n")
	}

	if len(report.FileFindings) > 0 {
		pathWidth := fileColumnWidth(report.FileFindings)
		buffer.printf("\n  No proof names these files (derived):\n")
		for _, finding := range report.FileFindings {
			if finding.IgnorePattern == "" {
				buffer.printf("    %s\n", finding.Path)
				continue
			}
			buffer.printf("    %-*s  [ignored: %s]\n", pathWidth, finding.Path, finding.IgnorePattern)
		}
	}

	buffer.printf("\n  Not instrumented: %d changed lines. No claim made.\n", report.UninstrumentedLines)
	return buffer.err
}

// lineColumnWidths sizes the two padded columns from the data.
func lineColumnWidths(findings []LineFinding) (int, int) {
	pathWidth, symbolWidth := 0, 0
	for _, finding := range findings {
		if len(finding.File) > pathWidth {
			pathWidth = len(finding.File)
		}
		if len(finding.Symbol) > symbolWidth {
			symbolWidth = len(finding.Symbol)
		}
	}
	return pathWidth, symbolWidth
}

func fileColumnWidth(findings []FileFinding) int {
	width := 0
	for _, finding := range findings {
		if len(finding.Path) > width {
			width = len(finding.Path)
		}
	}
	return width
}

// errorWriter keeps the first write error and stops writing after it.
type errorWriter struct {
	writer io.Writer
	err    error
}

func (writer *errorWriter) printf(format string, arguments ...any) {
	if writer.err != nil {
		return
	}
	_, writer.err = fmt.Fprintf(writer.writer, format, arguments...)
}
