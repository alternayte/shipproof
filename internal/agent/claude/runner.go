package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/alternayte/shipproof/internal/agent"
)

// printResult is the subset of the CLI JSON output that ShipProof reads.
// Every field is optional. A missing field stays missing.
type printResult struct {
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	IsError   bool   `json:"is_error"`
	Subtype   string `json:"subtype"`
}

// Run executes one bounded coding task. The reported status is a runner claim.
// It is never evidence.
func (runner Runner) Run(ctx context.Context, req agent.RunRequest) (agent.RunResult, error) {
	path, err := exec.LookPath(runner.binary)
	if err != nil {
		return agent.RunResult{}, fmt.Errorf("claude CLI not found: %w", err)
	}

	permissionMode := "plan"
	if req.Role == agent.RoleImplementer {
		permissionMode = "acceptEdits"
	}

	args := []string{"-p", "--output-format", "json", "--permission-mode", permissionMode}
	if runner.model != "" {
		args = append(args, "--model", runner.model)
	}
	args = append(args, agent.BuildPrompt(req))

	stdout, stderr, runErr := runCapture(ctx, path, req.Workspace, args...)

	result := agent.RunResult{
		Status: agent.RunStatusSuccess,
		Metadata: map[string]string{
			"transport":       "subprocess",
			"permission_mode": permissionMode,
			"exit_code":       "0",
		},
	}
	if runner.model != "" {
		result.Metadata["model"] = runner.model
	}

	if runErr != nil {
		result.Status = agent.RunStatusFailed
		exitErr, ok := runErr.(*exec.ExitError)
		if !ok {
			return agent.RunResult{}, fmt.Errorf("run claude: %w", runErr)
		}
		result.Metadata["exit_code"] = strconv.Itoa(exitErr.ExitCode())
	}

	var parsed printResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err == nil {
		result.SessionRef = parsed.SessionID
		result.Summary = strings.TrimSpace(parsed.Result)
		if parsed.IsError {
			result.Status = agent.RunStatusFailed
		}
		if parsed.Subtype != "" {
			result.Metadata["subtype"] = parsed.Subtype
		}
	} else {
		result.Summary = summarize(stdout, stderr)
	}

	return result, nil
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
