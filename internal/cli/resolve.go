package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/coverage"
	"github.com/alternayte/shipproof/internal/phase"
)

// soleOpenChange returns the single change that has not reached
// READY_FOR_HUMAN. With none or several, it returns an error and the list of
// open change identifiers.
func soleOpenChange(root string) (string, []string, error) {
	entries, err := os.ReadDir(filepath.Join(root, ".shipproof", "changes"))
	if err != nil {
		return "", nil, fmt.Errorf("no change exists; run `shipproof start <change-id> --intent <file>` first")
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

// printMatrix renders the matrix for a person.
func printMatrix(stdout io.Writer, matrix coverage.Matrix) {
	fmt.Fprintf(stdout, "Coverage — %s\n\n", matrix.ChangeID)

	width := 0
	for _, row := range matrix.Rows {
		if len(row.RequirementID) > width {
			width = len(row.RequirementID)
		}
	}

	counts := map[coverage.State]int{}
	for _, row := range matrix.Rows {
		counts[row.State]++
		fmt.Fprintf(stdout, "  %-*s  %-14s  %-8s  %s\n", width, row.RequirementID, row.State, row.Provenance, row.Detail)
	}

	fmt.Fprintln(stdout)
	for _, state := range []coverage.State{coverage.Proven, coverage.Failed, coverage.Accepted, coverage.AwaitingHuman, coverage.Unproven} {
		if counts[state] > 0 {
			fmt.Fprintf(stdout, "  %d %s\n", counts[state], state)
		}
	}
	if !matrix.RunCurrent {
		fmt.Fprintln(stdout, "\n  No proof result describes the working tree. Run `shipproof prove`.")
	}
}
