package codex

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/alternayte/shipproof/internal/agent"
)

// Run executes one bounded coding task. The reported status is a runner claim.
// It is never evidence. ShipProof verifies the repository itself.
func (runner Runner) Run(ctx context.Context, req agent.RunRequest) (agent.RunResult, error) {
	path, err := exec.LookPath(runner.binary)
	if err != nil {
		return agent.RunResult{}, fmt.Errorf("codex CLI not found: %w", err)
	}

	sandbox := "read-only"
	if req.Role == agent.RoleImplementer {
		sandbox = "workspace-write"
	}

	args := []string{"exec", "--skip-git-repo-check", "--sandbox", sandbox}
	if runner.model != "" {
		args = append(args, "--model", runner.model)
	}
	args = append(args, agent.BuildPrompt(req))

	stdout, stderr, runErr := runCapture(ctx, path, req.Workspace, args...)

	result := agent.RunResult{
		Status:  agent.RunStatusSuccess,
		Summary: summarize(stdout, stderr),
		Metadata: map[string]string{
			"transport": "subprocess",
			"sandbox":   sandbox,
		},
	}
	if runner.model != "" {
		result.Metadata["model"] = runner.model
	}
	if runErr != nil {
		result.Status = agent.RunStatusFailed
		var exitErr *exec.ExitError
		if ok := asExitError(runErr, &exitErr); ok {
			result.Metadata["exit_code"] = strconv.Itoa(exitErr.ExitCode())
		} else {
			return agent.RunResult{}, fmt.Errorf("run codex: %w", runErr)
		}
	} else {
		result.Metadata["exit_code"] = "0"
	}
	return result, nil
}

func asExitError(err error, target **exec.ExitError) bool {
	exitErr, ok := err.(*exec.ExitError)
	if ok {
		*target = exitErr
	}
	return ok
}

const summaryLimit = 4000

func summarize(stdout, stderr string) string {
	text := strings.TrimSpace(stdout)
	if text == "" {
		text = strings.TrimSpace(stderr)
	}
	if len(text) > summaryLimit {
		text = text[len(text)-summaryLimit:]
	}
	return text
}
