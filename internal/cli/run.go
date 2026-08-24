package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alternayte/shipproof/internal/agent"
	"github.com/alternayte/shipproof/internal/change"
	"github.com/alternayte/shipproof/internal/git"
	"github.com/alternayte/shipproof/internal/repository"
	"github.com/alternayte/shipproof/internal/verify"
)

// commandVerifier runs the repository verification command for one change.
type commandVerifier struct {
	root    string
	command string
}

func (verifier commandVerifier) Verify(ctx context.Context, changeID string) (agent.VerificationOutcome, error) {
	result, err := verify.Run(verifier.root, changeID, verifier.command)
	if err != nil {
		return agent.VerificationOutcome{}, err
	}
	detail := fmt.Sprintf("exit %d after %dms; see %s", result.ExitCode, result.DurationMs, result.StdoutPath)
	return agent.VerificationOutcome{Passed: result.ExitCode == 0, ExitCode: result.ExitCode, Detail: detail}, nil
}

// gitRevisions reads the real Git revision of the workspace.
type gitRevisions struct {
	root string
}

func (revisions gitRevisions) Revision(ctx context.Context) (string, error) {
	return git.HeadRevision(revisions.root)
}

// executorFactory builds the executor for one run. Tests replace it.
var executorFactory = buildExecutor

// runRun implements `shipproof run <change-id> [--runner <name>]`.
func runRun(args []string, stdout, stderr io.Writer) int {
	changeID := ""
	runnerOverride := ""
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--runner":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--runner requires a value")
				return 2
			}
			runnerOverride = args[index+1]
			index++
		default:
			if changeID != "" {
				fmt.Fprintln(stderr, "usage: shipproof run <change-id> [--runner <name>]")
				return 2
			}
			changeID = args[index]
		}
	}
	if changeID == "" {
		fmt.Fprintln(stderr, "usage: shipproof run <change-id> [--runner <name>]")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	record, err := change.Load(root, changeID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	executor, err := executorFactory(root, runnerOverride)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	request := agent.RunRequest{
		Workspace: root,
		Change: agent.Change{
			ID:           record.ChangeID,
			SnapshotPath: record.SnapshotPath,
			Intent:       readSnapshot(root, record.SnapshotPath),
		},
		Instructions: "Implement the approved change. Keep the change bounded to the approved scope.",
	}

	execution, err := executor.Execute(context.Background(), request)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	path, err := agent.SaveExecution(root, execution)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintf(stdout, "%s\n", execution.Status)
	if execution.Detail != "" {
		fmt.Fprintf(stdout, "detail: %s\n", execution.Detail)
	}
	fmt.Fprintf(stdout, "runner: %s\n", execution.Execution.Runner)
	fmt.Fprintf(stdout, "base revision: %s\n", execution.Execution.BaseRevision)
	fmt.Fprintf(stdout, "result revision: %s\n", execution.Execution.ResultRevision)
	fmt.Fprintf(stdout, "attempts: %d\n", len(execution.Attempts))
	for _, finding := range execution.Findings {
		fmt.Fprintf(stdout, "finding (%s, %s): %s\n", finding.Source, finding.Provenance, finding.Summary)
	}
	relative, relErr := filepath.Rel(root, path)
	if relErr != nil {
		relative = path
	}
	fmt.Fprintf(stdout, "execution record: %s\n", filepath.ToSlash(relative))

	switch execution.Status {
	case agent.OutcomePass:
		return 0
	case agent.OutcomeNeedsReview:
		return 1
	default:
		return 2
	}
}

// buildExecutor resolves the runner and assembles the executor.
func buildExecutor(root, runnerOverride string) (agent.Executor, error) {
	local, global, err := scopedAgentConfig(root)
	if err != nil {
		return agent.Executor{}, err
	}
	merged, err := repository.LoadAgentConfig(root)
	if err != nil {
		return agent.Executor{}, err
	}

	// Probe only when no explicit level selects a runner. A probe can be slow.
	selection, err := agent.Resolve(resolveInput(runnerOverride, nil, local, global))
	if errors.Is(err, agent.ErrNoUsableRunner) {
		results := probeAll(context.Background(), defaultRegistry, merged)
		selection, err = agent.Resolve(resolveInput(runnerOverride, usableNames(results), local, global))
	}
	if err != nil {
		return agent.Executor{}, fmt.Errorf("%w; run `shipproof runner doctor`", err)
	}

	runner, err := defaultRegistry.Build(agent.RunnerConfig{Name: selection.Name, Settings: merged.Runners[selection.Name]})
	if err != nil {
		return agent.Executor{}, err
	}

	executor := agent.Executor{
		Runner:      runner,
		RunnerName:  selection.Name,
		MaxAttempts: merged.RepairMaxAttempts,
		Revisions:   gitRevisions{root: root},
	}

	verifyConfig, err := verify.LoadConfig(root)
	if err != nil {
		return agent.Executor{}, err
	}
	executor.Verifier = commandVerifier{root: root, command: verifyConfig.Command}

	reviewName := merged.ReviewRunner
	if reviewName == "" {
		reviewName = selection.Name
	}
	reviewRunner, err := defaultRegistry.Build(agent.RunnerConfig{Name: reviewName, Settings: merged.Runners[reviewName]})
	if err != nil {
		return agent.Executor{}, err
	}
	executor.ReviewRunner = reviewRunner
	executor.ReviewRunnerName = reviewName

	return executor, nil
}

func readSnapshot(root, snapshotPath string) string {
	if strings.TrimSpace(snapshotPath) == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(snapshotPath)))
	if err != nil {
		return ""
	}
	return string(data)
}
