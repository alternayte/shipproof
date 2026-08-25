package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
	"github.com/alternayte/shipproof/internal/verify"
)

func runVerification(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof verification <init|check> ...")
		return 2
	}
	switch args[0] {
	case "run":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: shipproof verification run <change-id>")
			return 2
		}
		root, err := findRepositoryRoot(".")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		cfg, err := verify.LoadConfig(root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if _, err := change.Load(root, args[1]); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		result, err := verify.Run(root, args[1], cfg.Command)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		status := "passed"
		if result.ExitCode != 0 {
			status = "failed"
		}
		fmt.Fprintf(stdout, "Verification %s (exit %d, %dms)\n", status, result.ExitCode, result.DurationMs)
		fmt.Fprintf(stdout, "stdout: %s\n", result.StdoutPath)
		fmt.Fprintf(stdout, "stderr: %s\n", result.StderrPath)
		fmt.Fprintf(stdout, "result: .shipproof/runs/%s/run.json\n", result.ChangeID)
		return 0
	case "init":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: shipproof verification init <change-id>")
			return 2
		}
		root, err := findRepositoryRoot(".")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		path, err := verification.Initialize(root, args[1])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		rel, _ := filepath.Rel(root, path)
		fmt.Fprintf(stdout, "Created verification plan: %s\n", filepath.ToSlash(rel))
		fmt.Fprintln(stdout, "Next: use the plan-verification skill to map requirements and invariants to proof before implementation.")
		return 0
	case "check":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: shipproof verification check <change-id-or-file>")
			return 2
		}
		// The argument names the artifact that locates the repository. An
		// existing file resolves the root from its own path. A change
		// identifier resolves the root from the working directory.
		path := args[1]
		var root string
		var rootErr error
		if _, err := os.Stat(path); err == nil {
			root, rootErr = findRepositoryRoot(path)
		} else {
			root, rootErr = findRepositoryRoot(".")
			if rootErr != nil {
				fmt.Fprintln(stderr, rootErr)
				return 1
			}
			path = verification.Path(root, args[1])
		}
		plan, err := verification.Load(path)
		if err != nil {
			fmt.Fprintf(stderr, "invalid verification plan: %v\n", err)
			return 1
		}

		// The tie check runs only when a requirement sidecar exists. A change
		// adopted before the sidecar existed keeps the v0 behaviour.
		if rootErr == nil && requirements.Exists(root, plan.ChangeID) {
			set, err := requirements.Load(root, plan.ChangeID)
			if err != nil {
				fmt.Fprintf(stderr, "invalid requirement set: %v\n", err)
				return 1
			}
			blockers := verification.TieCheck(set, plan)
			if len(blockers) > 0 {
				for _, blocker := range blockers {
					fmt.Fprintf(stdout, "[BLOCKER] %s: %s\n", blocker.Kind, blocker.Detail)
				}
				fmt.Fprintf(stderr, "the requirement set and the verification plan disagree on %d identifier(s)\n", len(blockers))
				return 1
			}
			fmt.Fprintf(stdout, "Requirement tie check passed: %d requirements tied to a plan entry.\n", len(set.Requirements))
		}

		fmt.Fprintf(stdout, "Verification plan is valid: %s (%d requirements, %d invariants)\n", plan.ChangeID, len(plan.Requirements), len(plan.Invariants))
		return 0
	default:
		fmt.Fprintf(stderr, "unknown verification command %q\n", args[0])
		return 2
	}
}
