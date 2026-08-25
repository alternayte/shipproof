package cli

import (
	"fmt"
	"io"

	"github.com/alternayte/shipproof/internal/version"
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
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "harness":
		return runHarness(args[1:], stdout, stderr)
	case "change":
		return runChange(args[1:], stdout, stderr)
	case "next":
		return runNext(args[1:], stdout, stderr)
	case "coverage":
		return runCoverage(args[1:], stdout, stderr)
	case "plan":
		return runPlan(args[1:], stdout, stderr)
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
	case "report":
		return runReport(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "runner":
		return runRunner(args[1:], stdout, stderr)
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
	fmt.Fprintln(w, "  shipproof doc status <file> [--kind prd|sdd] [--json]")
	fmt.Fprintln(w, "  shipproof doc review <file> [--kind prd|sdd] [--json]")
	fmt.Fprintln(w, "  shipproof shape <prd|sdd|issue> <subject> [--id id] [--source path]")
	fmt.Fprintln(w, "  shipproof shape status <id> [--json]")
	fmt.Fprintln(w, "  shipproof shape check <id-or-file>")
	fmt.Fprintln(w, "  shipproof verification run <change-id> [--gate-only|--proofs-only]")
	fmt.Fprintln(w, "  shipproof verification init <change-id>")
	fmt.Fprintln(w, "  shipproof verification check <change-id-or-file>")
	fmt.Fprintln(w, "  shipproof verify [change-id]")
	fmt.Fprintln(w, "  shipproof harness install <claude|cursor|codex|opencode|agents> [directory] [--force] [--keep-retired]")
	fmt.Fprintln(w, "  shipproof change start <change-id> --source <path> [--ceremony 0|1|2|3]")
	fmt.Fprintln(w, "  shipproof change status <change-id>")
	fmt.Fprintln(w, "  shipproof change check <change-id>")
	fmt.Fprintln(w, "  shipproof next [change-id] [--json]")
	fmt.Fprintln(w, "  shipproof coverage <change-id> [--json]")
	fmt.Fprintln(w, "  shipproof plan create <file>")
	fmt.Fprintln(w, "  shipproof plan review")
	fmt.Fprintln(w, "  shipproof plan sync --linear [plan-file]")
	fmt.Fprintln(w, "  shipproof skill check [catalog-directory]")
	fmt.Fprintln(w, "  shipproof skill eval <check|list|show|record|results> ...")
	fmt.Fprintln(w, "  shipproof evidence pack <change-id> [--base <rev>] [--head <rev>]")
	fmt.Fprintln(w, "  shipproof evidence review <change-id>")
	fmt.Fprintln(w, "  shipproof telemetry collect <change-id> --adapter <claude|opencode> [--dir <path>]")
	fmt.Fprintln(w, "  shipproof review prepare <change-id>")
	fmt.Fprintln(w, "  shipproof linear issue <identifier>")
	fmt.Fprintln(w, "  shipproof linear project <name>")
	fmt.Fprintln(w, "  shipproof linear sync <plan-file>")
	fmt.Fprintln(w, "  shipproof report change <change-id> [--output path]")
	fmt.Fprintln(w, "  shipproof report pr-summary <change-id> [--output path]")
	fmt.Fprintln(w, "  shipproof report project <name> [--output path]")
	fmt.Fprintln(w, "  shipproof runner list")
	fmt.Fprintln(w, "  shipproof runner doctor")
	fmt.Fprintln(w, "  shipproof config get <key>")
	fmt.Fprintln(w, "  shipproof config set <key> <value> [--global|--local]")
	fmt.Fprintln(w, "  shipproof run <change-id> [--runner <name>]")
	fmt.Fprintln(w, "  shipproof version")
}
