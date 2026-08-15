package cli

import (
	"fmt"
	"io"

	"github.com/alternayte/shipproof/internal/verify"
)

// runVerify implements `shipproof verify`.
//
// With a change ID it behaves like `shipproof verification run`.
// Without a change ID it runs the configured verification command and stores
// logs under .shipproof/runs/adhoc/.
func runVerify(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "usage: shipproof verify [change-id]")
		return 2
	}
	if len(args) == 1 {
		return runVerification([]string{"run", args[0]}, stdout, stderr)
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cfg, err := verify.LoadConfig(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	result, err := verify.RunAdhoc(root, cfg.Command)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	status := "passed"
	if result.ExitCode != 0 {
		status = "failed"
	}
	fmt.Fprintf(stdout, "Verification %s (exit %d, %dms)\n", status, result.ExitCode, result.DurationMs)
	fmt.Fprintf(stdout, "stdout: %s\n", result.StdoutPath)
	fmt.Fprintf(stdout, "stderr: %s\n", result.StderrPath)

	return result.ExitCode
}
