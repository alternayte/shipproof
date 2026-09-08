package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/requirements"
	"github.com/alternayte/shipproof/internal/verification"
)

// runStart records the intent snapshot and adopts the requirement set. It
// replaces the old `change start` verb.
func runStart(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: shipproof start <change-id> --intent <path> [--ceremony 0|1|2|3] [--force]\n" +
		"       shipproof start <change-id> --confirm-requirements"

	if len(args) < 1 {
		fmt.Fprintln(stderr, usage)
		return 2
	}

	changeID := args[0]

	// `--confirm-requirements` adopts a standing proposal. It reads no intent
	// document, so it runs before every other option.
	for _, argument := range args[1:] {
		if argument != "--confirm-requirements" {
			continue
		}
		root, err := findRepositoryRoot(".")
		if err != nil {
			fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
			return 1
		}
		return confirmRequirements(root, changeID, stdout, stderr)
	}

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
			proposeRequirements(root, changeID, intent, stdout)
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

// proposeRequirements reads a document that a specification tool wrote. One
// documented pattern applies: a heading opens a requirement, and a list item
// that starts with MUST or SHALL states the obligation. Decision D3 fixes it.
//
// A pattern match is a proposal, never a fact. The proposal waits for a person
// to confirm it, and nothing adopts it in the meantime.
func proposeRequirements(root, changeID, intent string, stdout io.Writer) {
	body, err := os.ReadFile(intent)
	if err != nil {
		fmt.Fprintf(stdout, "Requirements: none adopted. %v\n", err)
		return
	}
	set, err := requirements.ProposeForeign(changeID, intent, body)
	if err != nil {
		fmt.Fprintln(stdout, "Requirements: none found. The document names no requirement identifier, and it states no obligation with MUST or SHALL.")
		return
	}
	if _, err := requirements.SaveProposal(root, set); err != nil {
		fmt.Fprintf(stdout, "Requirements: none proposed. %v\n", err)
		return
	}

	fmt.Fprintf(stdout, "Requirements: %d proposed, 0 adopted.\n", len(set.Requirements))
	for _, requirement := range set.Requirements {
		fmt.Fprintf(stdout, "  %s  %s\n", requirement.ID, requirement.Statement)
	}
	fmt.Fprintln(stdout, "Read each line. ShipProof matched a pattern; it judged nothing.")
	fmt.Fprintf(stdout, "Confirm them with:\n  shipproof start %s --confirm-requirements\n", changeID)
}

// confirmRequirements adopts the standing proposal. A person runs it, and the
// stamp records when.
func confirmRequirements(root, changeID string, stdout, stderr io.Writer) int {
	if requirements.Exists(root, changeID) {
		fmt.Fprintln(stderr, "the change already holds a requirement set")
		return 2
	}
	set, err := requirements.LoadProposal(root, changeID)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "no requirement proposal exists for %s\n", changeID)
			return 2
		}
		fmt.Fprintln(stderr, err)
		return 1
	}

	path, err := requirements.Save(root, set.Confirm(time.Now()))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := requirements.ClearProposal(root, changeID); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	rel, _ := filepath.Rel(root, path)
	fmt.Fprintf(stdout, "Requirements: confirmed %d into %s\n", len(set.Requirements), filepath.ToSlash(rel))
	fmt.Fprintf(stdout, "Next: run `shipproof status %s`.\n", changeID)
	return 0
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
