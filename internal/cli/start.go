package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		mergeRequirements(root, changeID, intent, stdout)
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

// mergeRequirements brings a standing requirement set up to date with the
// document. It adds what the document gained and keeps every requirement that
// stands, with the confirmation a person already gave.
//
// It never deletes. Principle 6 of Section 2 says nothing is deleted to make a
// change pass, so a requirement the document no longer states is reported and
// left in place. A person removes it deliberately or not at all.
func mergeRequirements(root, changeID, intent string, stdout io.Writer) {
	standing, err := requirements.Load(root, changeID)
	if err != nil {
		fmt.Fprintf(stdout, "Requirements: unchanged. %v\n", err)
		return
	}

	current, err := readDocumentRequirements(changeID, intent)
	if err != nil {
		fmt.Fprintln(stdout, "Requirements: unchanged. The document states none that ShipProof can read.")
		return
	}

	held := map[string]bool{}
	statements := map[string]bool{}
	for _, requirement := range standing.Requirements {
		held[requirement.ID] = true
		statements[requirement.Statement] = true
	}

	var added []string
	next := standing
	for _, candidate := range current.Requirements {
		// Match on the sentence, not the identifier. A document that gains a
		// line renumbers everything after it, and a renumbered requirement is
		// not a new one.
		if statements[candidate.Statement] {
			continue
		}
		candidate.ID = nextRequirementID(changeID, held)
		held[candidate.ID] = true
		statements[candidate.Statement] = true
		next.Requirements = append(next.Requirements, candidate)
		added = append(added, candidate.ID)
	}

	var dropped []string
	inDocument := map[string]bool{}
	for _, candidate := range current.Requirements {
		inDocument[candidate.Statement] = true
	}
	for _, requirement := range standing.Requirements {
		if !inDocument[requirement.Statement] {
			dropped = append(dropped, requirement.ID)
		}
	}

	if len(added) == 0 {
		fmt.Fprintln(stdout, "Requirements: the set already matches the document.")
	} else {
		// A merged requirement carries no confirmation, so Save refuses the
		// set until a person accepts the additions.
		if _, err := requirements.SaveProposal(root, next); err != nil {
			fmt.Fprintf(stdout, "Requirements: unchanged. %v\n", err)
			return
		}
		if _, err := requirements.Save(root, next.Confirm(time.Now())); err != nil {
			fmt.Fprintf(stdout, "Requirements: unchanged. %v\n", err)
			return
		}
		_ = requirements.ClearProposal(root, changeID)
		fmt.Fprintf(stdout, "Requirements: added %d from the document: %s\n",
			len(added), strings.Join(added, ", "))
	}

	if len(dropped) > 0 {
		fmt.Fprintf(stdout, "Requirements: the document no longer states %s. Nothing was deleted.\n",
			strings.Join(dropped, ", "))
		fmt.Fprintln(stdout, "              Remove it yourself if it is out of scope.")
	}
}

// readDocumentRequirements reads whichever pattern the document answers to.
func readDocumentRequirements(changeID, intent string) (requirements.Set, error) {
	if set, err := requirements.AdoptNative(changeID, intent); err == nil {
		return set, nil
	}
	body, err := os.ReadFile(intent)
	if err != nil {
		return requirements.Set{}, err
	}
	return requirements.ProposeForeign(changeID, intent, body)
}

// nextRequirementID picks the first identifier the set does not hold.
func nextRequirementID(changeID string, held map[string]bool) string {
	for index := 1; ; index++ {
		candidate := fmt.Sprintf("%s-R%d", changeID, index)
		if !held[candidate] {
			return candidate
		}
	}
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
