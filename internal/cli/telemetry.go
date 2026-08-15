package cli

import (
	"fmt"
	"io"

	"github.com/alternayte/shipproof/internal/telemetry"
)

func runTelemetry(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof telemetry <collect> ...")
		return 2
	}

	switch args[0] {
	case "collect":
		return runTelemetryCollect(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown telemetry command %q\n", args[0])
		return 2
	}
}

func runTelemetryCollect(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: shipproof telemetry collect <change-id> --adapter <claude|opencode> [--dir <path>]")
		return 2
	}

	changeID := args[0]
	adapterName := ""
	projectDir := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--adapter":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--adapter requires a value")
				return 2
			}
			i++
			adapterName = args[i]
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "--dir requires a path")
				return 2
			}
			i++
			projectDir = args[i]
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[i])
			return 2
		}
	}

	if adapterName == "" {
		fmt.Fprintln(stderr, "--adapter is required")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}

	if err := telemetry.Collect(root, changeID, adapterName, projectDir); err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Agent run record written: .shipproof/runs/%s/agent-run.json\n", changeID)
	return 0
}
