package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Result struct {
	SchemaVersion string `json:"schema_version"`
	ChangeID      string `json:"change_id"`
	ExitCode      int    `json:"exit_code"`
	DurationMs    int64  `json:"duration_ms"`
	StdoutPath    string `json:"stdout_path"`
	StderrPath    string `json:"stderr_path"`
	HeadRev       string `json:"head_rev,omitempty"`
	TreeClean     *bool  `json:"tree_clean,omitempty"`
	Timestamp     string `json:"timestamp"`
}

func RunDir(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "runs", changeID)
}

func Run(root, changeID, command string) (Result, error) {
	if strings.TrimSpace(changeID) == "" {
		return Result{}, fmt.Errorf("change id is required")
	}

	// Capture the tree state before the command runs. A verification command
	// can write files, so a state captured afterward would describe a tree that
	// the command itself changed.
	headRev, treeClean := TreeState(root)

	result, err := runCommand(root, RunDir(root, changeID), changeID, command)
	if err != nil {
		return Result{}, err
	}
	result.HeadRev = headRev
	result.TreeClean = treeClean

	resultPath := filepath.Join(RunDir(root, changeID), "run.json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("encode result: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(resultPath, data, 0o644); err != nil {
		return Result{}, fmt.Errorf("write result: %w", err)
	}

	return result, nil
}

// RunAdhoc executes the repository verification command without a change record.
// Logs go to .shipproof/runs/adhoc/. No run.json is written because no change
// exists to associate the result with.
func RunAdhoc(root, command string) (Result, error) {
	return runCommand(root, filepath.Join(root, ".shipproof", "runs", "adhoc"), "", command)
}

func runCommand(root, runDir, changeID, command string) (Result, error) {
	if strings.TrimSpace(command) == "" {
		return Result{}, ErrCommandMissing
	}

	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create run directory: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	start := time.Now()
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = root
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Env = os.Environ()

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return Result{}, fmt.Errorf("run command: %w", err)
		}
	}

	stdoutPath := filepath.Join(runDir, "stdout.log")
	if err := os.WriteFile(stdoutPath, stdoutBuf.Bytes(), 0o644); err != nil {
		return Result{}, fmt.Errorf("write stdout: %w", err)
	}

	stderrPath := filepath.Join(runDir, "stderr.log")
	if err := os.WriteFile(stderrPath, stderrBuf.Bytes(), 0o644); err != nil {
		return Result{}, fmt.Errorf("write stderr: %w", err)
	}

	relStdout, err := filepath.Rel(root, stdoutPath)
	if err != nil {
		relStdout = stdoutPath
	}
	relStderr, err := filepath.Rel(root, stderrPath)
	if err != nil {
		relStderr = stderrPath
	}

	return Result{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
		ExitCode:      exitCode,
		DurationMs:    duration.Milliseconds(),
		StdoutPath:    filepath.ToSlash(relStdout),
		StderrPath:    filepath.ToSlash(relStderr),
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}
