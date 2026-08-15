package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	ErrNoGitHubRemote = errors.New("repository origin is not a GitHub remote")
	ErrNoRemote       = errors.New("repository has no origin remote")
)

func ResolveGitHubRepo(dir string) (owner, name string, err error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", "", ErrNoRemote
	}

	url := strings.TrimSpace(string(out))
	if url == "" {
		return "", "", ErrNoRemote
	}

	owner, name, ok := parseGitHubURL(url)
	if !ok {
		return "", "", fmt.Errorf("%w: %s", ErrNoGitHubRemote, url)
	}

	return owner, name, nil
}

func parseGitHubURL(url string) (owner, name string, ok bool) {
	trimmed := strings.TrimSuffix(url, ".git")

	switch {
	case strings.HasPrefix(trimmed, "https://github.com/"):
		rest := strings.TrimPrefix(trimmed, "https://github.com/")
		return splitOwnerName(rest)
	case strings.HasPrefix(trimmed, "http://github.com/"):
		rest := strings.TrimPrefix(trimmed, "http://github.com/")
		return splitOwnerName(rest)
	case strings.HasPrefix(trimmed, "git@github.com:"):
		rest := strings.TrimPrefix(trimmed, "git@github.com:")
		return splitOwnerName(rest)
	default:
		return "", "", false
	}
}

func splitOwnerName(rest string) (owner, name string, ok bool) {
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
