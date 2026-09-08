package cli

import (
	"fmt"
	"io"

	"github.com/alternayte/shipproof/internal/version"
)

// foldedOrRemoved names every verb that no longer exists. Each one exits with
// code 2 and prints one line that names the replacement.
var foldedOrRemoved = []string{
	"doc", "shape", "plan", "linear",
	"verification", "verify", "harness", "change", "next",
	"coverage", "skill", "evidence", "review", "telemetry", "report",
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	for _, verb := range foldedOrRemoved {
		if args[0] == verb {
			return runRemoved(verb, stderr)
		}
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "start":
		return runStart(args[1:], stdout, stderr)
	case "prove":
		return runProve(args[1:], stdout, stderr)
	case "pack":
		return runPack(args[1:], stdout, stderr)
	case "status":
		return runStatus(args[1:], stdout, stderr)
	case "runner":
		return runRunner(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "shipproof %s\n", version.Version)
		return 0
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "ShipProof — evidence for AI-assisted software delivery")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  shipproof init [directory]")
	fmt.Fprintln(w, "  shipproof start <change-id> --intent <path> [--ceremony 0|1|2|3] [--force]")
	fmt.Fprintln(w, "  shipproof prove [change-id] [--gate-only|--proofs-only]")
	fmt.Fprintln(w, "  shipproof pack [change-id] [--base <rev>] [--head <rev>] [--adapter <name>] [--output <path>]")
	fmt.Fprintln(w, "  shipproof pack --verify <file>")
	fmt.Fprintln(w, "  shipproof status [change-id] [--json]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Support:")
	fmt.Fprintln(w, "  shipproof runner <list|doctor>")
	fmt.Fprintln(w, "  shipproof config get <key>")
	fmt.Fprintln(w, "  shipproof config set <key> <value> [--global|--local]")
	fmt.Fprintln(w, "  shipproof run <change-id> [--runner <name>]")
	fmt.Fprintln(w, "  shipproof version")
}
