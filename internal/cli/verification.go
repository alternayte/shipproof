package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/covprofile"
	"github.com/alternayte/shipproof/internal/proofs"
	"github.com/alternayte/shipproof/internal/repository"
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
		return runVerificationRun(args[1:], stdout, stderr)
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

// runVerificationRun implements
// `shipproof verification run <change-id> [--gate-only|--proofs-only]`.
//
// The command performs two jobs. The gate runs the repository verification
// command and stays the authority on whether the repository passes. The
// attribution pass runs each proof on its own and says which requirement a
// failure belongs to. Neither job replaces the other.
func runVerificationRun(args []string, stdout, stderr io.Writer) int {
	changeID := ""
	gateOnly := false
	proofsOnly := false
	for _, argument := range args {
		switch argument {
		case "--gate-only":
			gateOnly = true
		case "--proofs-only":
			proofsOnly = true
		default:
			if strings.HasPrefix(argument, "--") || changeID != "" {
				fmt.Fprintln(stderr, "usage: shipproof verification run <change-id> [--gate-only|--proofs-only]")
				return 2
			}
			changeID = argument
		}
	}
	if changeID == "" || (gateOnly && proofsOnly) {
		fmt.Fprintln(stderr, "usage: shipproof verification run <change-id> [--gate-only|--proofs-only]")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, err := change.Load(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	// Capture the tree state once, before either pass runs. A command can write
	// files, so a state captured afterwards describes a tree that the run itself
	// changed.
	//
	// run.json and proofs.json take separate captures. The two agree only while
	// nothing between the two calls writes to the tree.
	headRev, treeClean := verify.TreeState(root)

	if !proofsOnly {
		cfg, err := verify.LoadConfig(root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		result, err := verify.Run(root, changeID, cfg.Command)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		status := "passed"
		if result.ExitCode != 0 {
			status = "failed"
		}
		fmt.Fprintf(stdout, "Gate %s (exit %d, %dms)\n", status, result.ExitCode, result.DurationMs)
		fmt.Fprintf(stdout, "stdout: %s\n", result.StdoutPath)
		fmt.Fprintf(stdout, "stderr: %s\n", result.StderrPath)
		fmt.Fprintf(stdout, "result: .shipproof/runs/%s/run.json\n", changeID)
	}

	if gateOnly {
		return 0
	}

	planPath := verification.Path(root, changeID)
	if _, err := os.Stat(planPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stdout, "No proof ran: no verification plan exists for %s.\n", changeID)
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	plan, err := verification.Load(planPath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid verification plan: %v\n", err)
		return 1
	}

	var coverageOptions *proofs.Coverage
	staleCoverageDir := false
	if cfg, err := repository.LoadConfig(root); err == nil {
		if template := strings.TrimSpace(cfg.Verification.Coverage.Command); template != "" {
			coverageOptions = &proofs.Coverage{
				Command: template,
				Dir:     proofs.CoverageDir(root, changeID),
			}
			if err := clearCoverageDir(coverageOptions.Dir); err != nil {
				fmt.Fprintf(stderr, "coverage: could not clear the coverage directory, a stale profile can survive: %v\n", err)
				staleCoverageDir = true
			}
		}
	}

	set, err := proofs.Run(root, plan, headRev, treeClean, coverageOptions)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, err := proofs.Save(root, set); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	passed, failed, human := 0, 0, 0
	for _, result := range set.Results {
		switch result.Status {
		case proofs.Pass:
			passed++
		case proofs.Fail:
			failed++
			fmt.Fprintf(stdout, "  fail %s proof %d: %s (exit %d)\n", result.RequirementID, result.ProofIndex, result.Command, result.ExitCode)
		case proofs.Human:
			human++
		}
	}
	fmt.Fprintf(stdout, "Proofs: %d passed, %d failed, %d human\n", passed, failed, human)
	fmt.Fprintf(stdout, "results: .shipproof/runs/%s/proofs.json\n", changeID)

	if coverageOptions != nil {
		merged, count := mergeProfiles(root, changeID, coverageOptions.Dir)
		if merged != "" {
			note := ""
			if staleCoverageDir {
				note = " (a stale profile from an earlier revision may be present)"
			}
			fmt.Fprintf(stdout, "coverage: %s (%d profiles)%s\n", merged, count, note)
		}
	}
	return 0
}

// clearCoverageDir removes a stale coverage directory from an earlier
// revision. The caller must not treat a failure here as a run failure. The
// signal never blocks a run, so measurement still runs against whatever
// profiles remain, and a surviving stale profile is the lesser harm.
func clearCoverageDir(dir string) error {
	return os.RemoveAll(dir)
}

// mergeProfiles joins every per-proof profile into one. It returns the
// repository-relative merged path and the count of profiles it read. A failure
// returns an empty path, because coverage never fails a run.
func mergeProfiles(root, changeID, dir string) (string, int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0
	}
	modulePath := covprofile.ModulePath(root)
	merged := &strings.Builder{}
	merged.WriteString("mode: set\n")
	count := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".out") || name == "merged.out" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if _, err := covprofile.Parse(strings.NewReader(string(data)), modulePath); err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "mode:") {
				continue
			}
			merged.WriteString(trimmed)
			merged.WriteString("\n")
		}
		count++
	}
	if count == 0 {
		return "", 0
	}
	path := proofs.MergedProfilePath(root, changeID)
	if err := os.WriteFile(path, []byte(merged.String()), 0o644); err != nil {
		return "", 0
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path, count
	}
	return filepath.ToSlash(relative), count
}
