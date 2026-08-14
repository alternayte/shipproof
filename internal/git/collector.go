package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Commit struct {
	Hash       string `json:"hash"`
	Author     string `json:"author"`
	Timestamp  string `json:"timestamp"`
	Subject    string `json:"subject"`
	Provenance string `json:"provenance"`
}

type Metadata struct {
	Commits      []Commit `json:"commits"`
	ChangedFiles []string `json:"changed_files"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	DiffStat     string   `json:"diff_stat"`
	Provenance   string   `json:"provenance"`
}

func CollectMetadata(dir, base, head string) (Metadata, error) {
	if err := checkRepo(dir); err != nil {
		return Metadata{}, err
	}

	md := Metadata{
		Provenance: "observed",
	}

	commits, err := CollectCommits(dir, base, head)
	if err != nil {
		return Metadata{}, fmt.Errorf("collect commits: %w", err)
	}
	md.Commits = commits

	files, err := CollectChangedFiles(dir, base, head)
	if err != nil {
		return Metadata{}, fmt.Errorf("collect changed files: %w", err)
	}
	md.ChangedFiles = files

	adds, dels, err := CountLines(dir, base, head)
	if err != nil {
		return Metadata{}, fmt.Errorf("count lines: %w", err)
	}
	md.Additions = adds
	md.Deletions = dels

	stat, err := DiffStat(dir, base, head)
	if err != nil {
		return Metadata{}, fmt.Errorf("diff stat: %w", err)
	}
	md.DiffStat = stat

	return md, nil
}

func CollectCommits(dir, base, head string) ([]Commit, error) {
	if err := checkRepo(dir); err != nil {
		return nil, err
	}
	rangeStr := base + ".." + head
	cmd := exec.Command("git", "log", "--format=format:%H%x00%an%x00%aI%x00%s", rangeStr)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		if isBadRevision(err) {
			return nil, fmt.Errorf("revision range %q: %w", rangeStr, ErrBadRevision)
		}
		return nil, fmt.Errorf("git log: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	var commits []Commit
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, Commit{
			Hash:       parts[0],
			Author:     parts[1],
			Timestamp:  parts[2],
			Subject:    parts[3],
			Provenance: "observed",
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read git log output: %w", err)
	}

	return commits, nil
}

func CollectChangedFiles(dir, base, head string) ([]string, error) {
	if err := checkRepo(dir); err != nil {
		return nil, err
	}
	rangeStr := base + ".." + head
	cmd := exec.Command("git", "diff", "--name-only", rangeStr)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		if isBadRevision(err) {
			return nil, fmt.Errorf("revision range %q: %w", rangeStr, ErrBadRevision)
		}
		return nil, fmt.Errorf("git diff --name-only: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	var files []string
	for scanner.Scan() {
		name := scanner.Text()
		if name == "" {
			continue
		}
		files = append(files, name)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read git diff output: %w", err)
	}

	return files, nil
}

func CountLines(dir, base, head string) (additions, deletions int, err error) {
	if err := checkRepo(dir); err != nil {
		return 0, 0, err
	}
	rangeStr := base + ".." + head
	cmd := exec.Command("git", "diff", "--shortstat", rangeStr)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		if isBadRevision(err) {
			return 0, 0, fmt.Errorf("revision range %q: %w", rangeStr, ErrBadRevision)
		}
		return 0, 0, fmt.Errorf("git diff --shortstat: %w", err)
	}

	stat := strings.TrimSpace(string(out))
	if stat == "" {
		return 0, 0, nil
	}

	additions, deletions = parseShortStat(stat)
	return additions, deletions, nil
}

func DiffStat(dir, base, head string) (string, error) {
	if err := checkRepo(dir); err != nil {
		return "", err
	}
	rangeStr := base + ".." + head
	cmd := exec.Command("git", "diff", "--stat", rangeStr)
	cmd.Dir = dir

	out, err := cmd.Output()
	if err != nil {
		if isBadRevision(err) {
			return "", fmt.Errorf("revision range %q: %w", rangeStr, ErrBadRevision)
		}
		return "", fmt.Errorf("git diff --stat: %w", err)
	}

	return string(out), nil
}

func checkRepo(dir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrGitNotInstalled
	}
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", dir, ErrNotGitRepo)
	}
	return nil
}

func isBadRevision(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 128
	}
	return false
}

func parseShortStat(stat string) (additions, deletions int) {
	insertionsIdx := strings.Index(stat, "insertion")
	if insertionsIdx >= 0 {
		before := strings.TrimSpace(stat[:insertionsIdx])
		lastComma := strings.LastIndex(before, ",")
		if lastComma >= 0 {
			numStr := strings.TrimSpace(before[lastComma+1:])
			n, err := strconv.Atoi(numStr)
			if err == nil {
				additions = n
			}
		} else if fields := strings.Fields(before); len(fields) > 0 {
			n, err := strconv.Atoi(fields[0])
			if err == nil {
				additions = n
			}
		}
	}

	deletionsIdx := strings.Index(stat, "deletion")
	if deletionsIdx >= 0 {
		before := strings.TrimSpace(stat[:deletionsIdx])
		lastComma := strings.LastIndex(before, ",")
		if lastComma >= 0 {
			numStr := strings.TrimSpace(before[lastComma+1:])
			n, err := strconv.Atoi(numStr)
			if err == nil {
				deletions = n
			}
		} else if fields := strings.Fields(before); len(fields) > 0 {
			n, err := strconv.Atoi(fields[0])
			if err == nil {
				deletions = n
			}
		}
	}

	return additions, deletions
}
