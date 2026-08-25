package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/phase"
)

func runNext(args []string, stdout, stderr io.Writer) int {
	var changeID string
	asJSON := false

	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--json" {
			asJSON = true
			continue
		}
		if strings.HasPrefix(argument, "--") {
			fmt.Fprintf(stderr, "unknown option %q\n", argument)
			return 2
		}
		if changeID != "" {
			fmt.Fprintf(stderr, "unexpected argument %q\n", argument)
			return 2
		}
		changeID = argument
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if changeID == "" {
		resolved, open, err := soleOpenChange(root)
		if err != nil {
			fmt.Fprintln(stderr, err)
			for _, candidate := range open {
				fmt.Fprintf(stderr, "  %s\n", candidate)
			}
			return 2
		}
		changeID = resolved
	}

	result, err := phase.Resolve(root, changeID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if asJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode result: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", data)
		return 0
	}

	fmt.Fprintf(stdout, "change    %s\n", result.ChangeID)
	fmt.Fprintf(stdout, "phase     %s\n", result.Phase)
	if result.Blocker != "" {
		fmt.Fprintf(stdout, "blocker   %s\n", result.Blocker)
	}
	if result.NextCommand != "" {
		fmt.Fprintf(stdout, "next      %s\n", result.NextCommand)
	}
	if result.NextSkill != "" {
		fmt.Fprintf(stdout, "skill     %s\n", result.NextSkill)
	}
	return 0
}

// soleOpenChange returns the single change that has not reached
// READY_FOR_HUMAN. With none or several, it returns an error and the list of
// open change identifiers.
func soleOpenChange(root string) (string, []string, error) {
	entries, err := os.ReadDir(filepath.Join(root, ".shipproof", "changes"))
	if err != nil {
		return "", nil, fmt.Errorf("no change exists; run shipproof change start first")
	}

	var open []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// A directory with no change.json is not a change. shipproof
		// verification init creates such a directory, and counting it would
		// report a phantom open change forever. Any other stat failure is a
		// real fault. Dropping the change here would hide it.
		if _, err := os.Stat(change.Path(root, entry.Name())); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", nil, fmt.Errorf("inspect change %s: %w", entry.Name(), err)
		}
		result, err := phase.Resolve(root, entry.Name())
		if err != nil {
			// A malformed artifact is an error, never a phase. Dropping the
			// change here would hide the corruption.
			return "", nil, fmt.Errorf("resolve change %s: %w", entry.Name(), err)
		}
		if result.Phase != phase.ReadyForHuman {
			open = append(open, entry.Name())
		}
	}
	sort.Strings(open)

	switch len(open) {
	case 0:
		return "", nil, fmt.Errorf("no open change exists; name a change identifier")
	case 1:
		return open[0], nil, nil
	default:
		return "", open, fmt.Errorf("%d open changes; name one:", len(open))
	}
}
