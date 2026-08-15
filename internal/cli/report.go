package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/shipproof/shipproof/internal/change"
	"github.com/shipproof/shipproof/internal/report"
)

func runReport(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof report <change|pr-summary> ...")
		return 2
	}

	switch args[0] {
	case "change":
		return runReportChange(args[1:], stdout, stderr)
	case "pr-summary":
		return runReportPRSummary(args[1:], stdout, stderr)
	case "project":
		return runReportProject(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown report command %q\n", args[0])
		return 2
	}
}

func runReportChange(args []string, stdout, stderr io.Writer) int {
	var outputPath string
	changeID, ok := parseReportArgs(args, &outputPath, stderr)
	if !ok {
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if _, err := change.Load(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	var writer io.Writer = stdout
	var closer io.Closer
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		f, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		writer = f
		closer = f
		defer f.Close()
	}

	if err := report.GenerateChangeReport(writer, root, changeID); err != nil {
		if closer != nil {
			closer.Close()
		}
		fmt.Fprintln(stderr, err)
		return 1
	}

	if outputPath != "" {
		abs, _ := filepath.Abs(outputPath)
		fmt.Fprintf(stdout, "Change report written: %s\n", abs)
	}

	return 0
}

func runReportPRSummary(args []string, stdout, stderr io.Writer) int {
	var outputPath string
	changeID, ok := parseReportArgs(args, &outputPath, stderr)
	if !ok {
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if _, err := change.Load(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	var writer io.Writer = stdout
	var closer io.Closer
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		f, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		writer = f
		closer = f
		defer f.Close()
	}

	if err := report.GeneratePRSummary(writer, root, changeID); err != nil {
		if closer != nil {
			closer.Close()
		}
		fmt.Fprintln(stderr, err)
		return 1
	}

	if outputPath != "" {
		abs, _ := filepath.Abs(outputPath)
		fmt.Fprintf(stdout, "PR summary written: %s\n", abs)
	}

	return 0
}

func runReportProject(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof report project <name> [--output path]")
		return 2
	}

	projectName := args[0]
	var outputPath string

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--output requires a path")
				return 2
			}
			outputPath = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[i])
			return 2
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	var writer io.Writer = stdout
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		f, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		writer = f
		defer f.Close()
	}

	if err := report.GenerateProjectReport(writer, root, projectName); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if outputPath != "" {
		abs, _ := filepath.Abs(outputPath)
		fmt.Fprintf(stdout, "Project report written: %s\n", abs)
	}

	return 0
}

func parseReportArgs(args []string, outputPath *string, stderr io.Writer) (string, bool) {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof report change <change-id> [--output path]")
		return "", false
	}

	changeID := args[0]

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--output requires a path")
				return "", false
			}
			*outputPath = args[i+1]
			i++
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[i])
			return "", false
		}
	}

	if changeID == "" {
		fmt.Fprintln(stderr, "change-id is required")
		return "", false
	}

	return changeID, true
}
