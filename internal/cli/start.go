package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
)

// runStart records the intent snapshot and adopts the requirement set. It
// replaces the old `change start` verb.
func runStart(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: shipproof start <change-id> --intent <path> [--ceremony 0|1|2|3] [--force]"

	if len(args) < 1 {
		fmt.Fprintln(stderr, usage)
		return 2
	}

	changeID := args[0]
	var intent string
	var ceremony *int
	force := false
	for index := 1; index < len(args); index++ {
		switch args[index] {
		case "--intent":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--intent requires a path")
				return 2
			}
			intent = args[index+1]
			index++
		case "--ceremony":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--ceremony requires a level from 0 to 3")
				return 2
			}
			parsed, err := strconv.Atoi(args[index+1])
			if err != nil || parsed < 0 || parsed > change.MaxCeremony {
				fmt.Fprintf(stderr, "--ceremony requires a level from 0 to %d; got %q\n", change.MaxCeremony, args[index+1])
				return 2
			}
			ceremony = &parsed
			index++
		case "--force":
			force = true
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[index])
			return 2
		}
	}

	if intent == "" {
		fmt.Fprintln(stderr, "--intent is required")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	var record change.Record
	if force {
		record, err = change.Restart(root, changeID, intent, ceremony)
	} else {
		level := change.DefaultCeremony
		if ceremony != nil {
			level = *ceremony
		}
		record, err = change.Start(root, changeID, intent, level)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	rel, _ := filepath.Rel(root, change.Path(root, changeID))
	fmt.Fprintf(stdout, "Started change %s\n", changeID)
	fmt.Fprintf(stdout, "Intent: %s\n", record.SourcePath)
	fmt.Fprintf(stdout, "Snapshot: %s\n", record.SnapshotPath)
	fmt.Fprintf(stdout, "SHA-256: %s\n", record.SHA256)
	fmt.Fprintf(stdout, "Ceremony: %d\n", record.CeremonyLevel())
	fmt.Fprintf(stdout, "Captured: %s\n", record.CapturedAt)
	fmt.Fprintf(stdout, "Record: %s\n", filepath.ToSlash(rel))
	adoptRequirements(root, changeID, intent, stdout)
	scaffoldPlan(root, changeID, stdout)
	fmt.Fprintf(stdout, "Next: run `shipproof status %s`.\n", changeID)
	return 0
}

// adoptRequirements writes the requirement set when the intent document holds
// one. A document that holds none is not an error. The count stays honest.
func adoptRequirements(root, changeID, intent string, stdout io.Writer) {
	if requirements.Exists(root, changeID) {
		fmt.Fprintln(stdout, "Requirements: the change already holds a requirement set.")
		return
	}

	set, err := requirements.AdoptNative(changeID, intent)
	if err != nil {
		if errors.Is(err, requirements.ErrNoNativeRequirement) {
			fmt.Fprintln(stdout, "Requirements: none found. The intent document names no requirement identifier.")
			return
		}
		fmt.Fprintf(stdout, "Requirements: none adopted. %v\n", err)
		return
	}

	path, err := requirements.Save(root, set)
	if err != nil {
		fmt.Fprintf(stdout, "Requirements: none adopted. %v\n", err)
		return
	}
	rel, _ := filepath.Rel(root, path)
	fmt.Fprintf(stdout, "Requirements: adopted %d into %s\n", len(set.Requirements), filepath.ToSlash(rel))
}

// scaffoldPlan writes an empty verification plan when the change holds none.
// The plan is the file the agent fills with one proof per requirement. An
// existing plan stays untouched.
func scaffoldPlan(root, changeID string, stdout io.Writer) {
	if _, err := os.Stat(verification.Path(root, changeID)); err == nil {
		fmt.Fprintln(stdout, "Plan: the change already holds a verification plan.")
		return
	}
	path, err := verification.Initialize(root, changeID)
	if err != nil {
		fmt.Fprintf(stdout, "Plan: none created. %v\n", err)
		return
	}
	rel, _ := filepath.Rel(root, path)
	fmt.Fprintf(stdout, "Plan: created %s. Map one proof to each requirement.\n", filepath.ToSlash(rel))
}
