package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/shipproof/shipproof/internal/plan"
)

func runPlan(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof plan <create|review|sync> ...")
		return 2
	}

	switch args[0] {
	case "create":
		return runPlanCreate(args[1:], stdout, stderr)
	case "review":
		return runPlanReview(args[1:], stdout, stderr)
	case "sync":
		return runPlanSync(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown plan command %q\n", args[0])
		return 2
	}
}

func runPlanCreate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof plan create <file>")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	record, err := plan.Create(root, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	rel, _ := filepath.Rel(root, plan.Path(root, record.PlanID))
	fmt.Fprintf(stdout, "Created plan %s\n", record.PlanID)
	fmt.Fprintf(stdout, "Source: %s\n", record.SourcePath)
	fmt.Fprintf(stdout, "Snapshot: %s\n", record.SnapshotPath)
	fmt.Fprintf(stdout, "SHA-256: %s\n", record.SHA256)
	fmt.Fprintf(stdout, "Record: %s\n", filepath.ToSlash(rel))
	fmt.Fprintln(stdout, "Next: decompose the plan into changes with the decompose-plan skill.")
	return 0
}

func runPlanReview(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: shipproof plan review")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	ids, err := plan.List(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if len(ids) == 0 {
		fmt.Fprintln(stdout, "No plans found under .shipproof/plans/")
		return 0
	}

	failed := false
	for _, id := range ids {
		record, err := plan.Load(root, id)
		if err != nil {
			fmt.Fprintf(stdout, "%s: invalid record: %v\n", id, err)
			failed = true
			continue
		}
		if err := record.VerifyHash(root); err != nil {
			fmt.Fprintf(stdout, "%s: %v\n", id, err)
			failed = true
			continue
		}
		stale, current, err := record.Staleness(root)
		if err != nil {
			fmt.Fprintf(stdout, "%s: %v\n", id, err)
			failed = true
			continue
		}
		if stale {
			fmt.Fprintf(stdout, "%s: valid, but the source changed since the snapshot (current %s)\n", id, current)
			continue
		}
		fmt.Fprintf(stdout, "%s: valid\n", id)
	}

	if failed {
		return 1
	}
	return 0
}

func runPlanSync(args []string, stdout, stderr io.Writer) int {
	planFile := ""
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--linear":
		default:
			if planFile != "" {
				fmt.Fprintln(stderr, "usage: shipproof plan sync --linear [plan-file]")
				return 2
			}
			planFile = args[index]
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if planFile == "" {
		found, err := findIssuesFile(root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		planFile = found
	}

	return syncPlanFile(planFile, stdout, stderr)
}

// findIssuesFile locates the Linear issue list for a plan. It accepts exactly
// one issues.json under .shipproof/plans/.
func findIssuesFile(root string) (string, error) {
	var matches []string
	err := filepath.WalkDir(plan.Dir(root), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == "issues.json" {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("scan plans directory: %w", err)
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no issues.json found under .shipproof/plans/; pass a plan file or create one with the decompose-plan skill")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple issues.json files found; pass one explicitly: %v", matches)
	}
}
