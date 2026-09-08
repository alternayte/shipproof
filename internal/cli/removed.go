package cli

import (
	"fmt"
	"io"
)

// removedMessages holds one line for each verb that SP-023 cut. The line
// names the replacement, or it states that the feature is gone.
var removedMessages = map[string]string{
	"doc":    "shipproof doc is gone. Document quality review is not a ShipProof feature.",
	"shape":  "shipproof shape is gone. Use your spec tool, then run `shipproof change start <change-id> --source <file>`.",
	"plan":   "shipproof plan is gone. Plan decomposition is not a ShipProof feature.",
	"linear": "shipproof linear is gone. ShipProof needs no issue tracker credential.",
}

// runRemoved prints the one line for a cut verb and returns exit code 2.
func runRemoved(verb string, stderr io.Writer) int {
	fmt.Fprintln(stderr, removedMessages[verb])
	return 2
}
