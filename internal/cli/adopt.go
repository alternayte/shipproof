package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alternayte/shipproof/internal/requirements"
)

// adoptNow supplies the confirmation stamp. A test replaces it to keep output
// deterministic.
var adoptNow = func() time.Time { return time.Now() }

func runDocAdopt(args []string, stdout, stderr io.Writer) int {
	const usage = "usage: shipproof doc adopt <change-id> --source <path> [--confirm] [--json]"

	if len(args) < 1 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	changeID := args[0]

	source := ""
	confirm := false
	jsonOutput := false
	for index := 1; index < len(args); index++ {
		switch args[index] {
		case "--source":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--source requires a path")
				return 2
			}
			source = args[index+1]
			index++
		case "--confirm":
			confirm = true
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintf(stderr, "unknown option %q\n%s\n", args[index], usage)
			return 2
		}
	}
	if source == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}

	body, err := os.ReadFile(source)
	if err != nil {
		fmt.Fprintf(stderr, "read source document: %v\n", err)
		return 1
	}

	root, err := findRepositoryRoot(source)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	set, err := requirements.ParseNative(changeID, relativeSource(root, source), body)
	if err == nil {
		return writeAdoptedSet(root, set, jsonOutput, stdout, stderr)
	}
	if !isNoNativeRequirement(err) {
		fmt.Fprintf(stderr, "parse source document: %v\n", err)
		return 1
	}

	proposal, err := requirements.ProposeForeign(changeID, relativeSource(root, source), body)
	if err != nil {
		fmt.Fprintf(stderr, "propose a requirement set: %v\n", err)
		return 1
	}

	if !confirm {
		printProposal(stdout, proposal)
		fmt.Fprintf(stderr, "%s is not a native ShipProof document. Review the proposal above, then rerun with --confirm.\n", source)
		return 1
	}

	return writeAdoptedSet(root, proposal.Confirm(adoptNow()), jsonOutput, stdout, stderr)
}

func writeAdoptedSet(root string, set requirements.Set, jsonOutput bool, stdout, stderr io.Writer) int {
	path, err := requirements.Save(root, set)
	if err != nil {
		fmt.Fprintf(stderr, "write requirement set: %v\n", err)
		return 1
	}
	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(set); err != nil {
			fmt.Fprintf(stderr, "write requirement set: %v\n", err)
			return 1
		}
		return 0
	}
	rel, relErr := filepath.Rel(root, path)
	if relErr != nil {
		rel = path
	}
	fmt.Fprintf(stdout, "Adopted %d requirements from the %s adopter: %s\n",
		len(set.Requirements), set.Adopter, filepath.ToSlash(rel))
	fmt.Fprintln(stdout, "Next: run shipproof verification check to tie every requirement to a proof.")
	return 0
}

func printProposal(w io.Writer, set requirements.Set) {
	fmt.Fprintf(w, "Proposed requirement set for %s (%d candidates):\n\n", set.ChangeID, len(set.Requirements))
	for _, requirement := range set.Requirements {
		fmt.Fprintf(w, "  %s  %s\n", requirement.ID, requirement.Statement)
	}
	fmt.Fprintln(w)
}

// relativeSource records the source path relative to the repository root, so
// the sidecar stays portable across machines.
func relativeSource(root, source string) string {
	abs, err := filepath.Abs(source)
	if err != nil {
		return source
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return source
	}
	return filepath.ToSlash(rel)
}

func isNoNativeRequirement(err error) bool {
	return errors.Is(err, requirements.ErrNoNativeRequirement)
}
