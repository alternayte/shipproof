package cli

import (
	"fmt"
	"io"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "doc":
		return runDoc(args[1:], stdout, stderr)
	case "shape":
		return runShape(args[1:], stdout, stderr)
	case "verification":
		return runVerification(args[1:], stdout, stderr)
	case "harness":
		return runHarness(args[1:], stdout, stderr)
	case "change":
		return runChange(args[1:], stdout, stderr)
	case "skill":
		return runSkill(args[1:], stdout, stderr)
	case "evidence":
		return runEvidence(args[1:], stdout, stderr)
	case "review":
		return runReview(args[1:], stdout, stderr)
	case "linear":
		return runLinear(args[1:], stdout, stderr)
	case "telemetry":
		return runTelemetry(args[1:], stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, "shipproof 0.2.0-dev")
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
	fmt.Fprintln(w, "  shipproof doc status <file> [--kind prd|sdd] [--json]")
	fmt.Fprintln(w, "  shipproof doc review <file> [--kind prd|sdd] [--json]")
	fmt.Fprintln(w, "  shipproof shape <prd|sdd|issue> <subject> [--id id] [--source path]")
	fmt.Fprintln(w, "  shipproof shape status <id> [--json]")
	fmt.Fprintln(w, "  shipproof shape check <id-or-file>")
	fmt.Fprintln(w, "  shipproof verification run <change-id>")
	fmt.Fprintln(w, "  shipproof verification init <change-id>")
	fmt.Fprintln(w, "  shipproof verification check <change-id-or-file>")
	fmt.Fprintln(w, "  shipproof harness install <claude|cursor|codex|opencode|agents> [directory] [--force]")
	fmt.Fprintln(w, "  shipproof change start <change-id> --source <path>")
	fmt.Fprintln(w, "  shipproof change status <change-id>")
	fmt.Fprintln(w, "  shipproof change check <change-id>")
	fmt.Fprintln(w, "  shipproof skill check [catalog-directory]")
	fmt.Fprintln(w, "  shipproof skill eval <check|list|show> ...")
	fmt.Fprintln(w, "  shipproof evidence pack <change-id> [--base <rev>] [--head <rev>]")
	fmt.Fprintln(w, "  shipproof telemetry collect <change-id> --adapter <claude|opencode> [--dir <path>]")
	fmt.Fprintln(w, "  shipproof review prepare <change-id>")
	fmt.Fprintln(w, "  shipproof linear issue <identifier>")
	fmt.Fprintln(w, "  shipproof linear project <name>")
	fmt.Fprintln(w, "  shipproof linear sync <plan-file>")
	fmt.Fprintln(w, "  shipproof version")
}
