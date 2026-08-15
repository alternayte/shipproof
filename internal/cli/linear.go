package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/shipproof/shipproof/internal/linear"
)

func runLinear(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof linear <issue|project|sync> ...")
		return 2
	}

	switch args[0] {
	case "issue":
		return runLinearIssue(args[1:], stdout, stderr)
	case "project":
		return runLinearProject(args[1:], stdout, stderr)
	case "sync":
		return runLinearSync(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown linear command %q\n", args[0])
		return 2
	}
}

func runLinearIssue(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof linear issue <identifier>")
		return 2
	}

	apiKey, err := linear.ResolveAPIKey("")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	client, err := linear.NewClient(apiKey)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	issue, err := linear.GetIssue(client, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(issue); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runLinearProject(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof linear project <name>")
		return 2
	}

	apiKey, err := linear.ResolveAPIKey("")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	client, err := linear.NewClient(apiKey)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	project, err := linear.GetProject(client, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(project); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runLinearSync(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof linear sync <plan-file>")
		return 2
	}
	return syncPlanFile(args[0], stdout, stderr)
}

func syncPlanFile(planFile string, stdout, stderr io.Writer) int {
	apiKey, err := linear.ResolveAPIKey("")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	client, err := linear.NewClient(apiKey)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	teamID := os.Getenv("LINEAR_TEAM_ID")
	if teamID == "" {
		fmt.Fprintln(stderr, "LINEAR_TEAM_ID is not set; export LINEAR_TEAM_ID")
		return 1
	}

	result, err := linear.SyncPlan(client, planFile, teamID, os.Stdin, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
