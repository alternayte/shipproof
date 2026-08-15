package github

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrNoToken    = errors.New("GITHUB_TOKEN is not set; export GITHUB_TOKEN with read access to the repository")
	ErrAuthFailed = errors.New("authentication failed; check your GITHUB_TOKEN")
	ErrRateLimit  = errors.New("GitHub API rate limit exceeded; wait before retrying")
	ErrNotFound   = errors.New("resource not found on GitHub")
)

func ResolveToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	return "", ErrNoToken
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	switch e.StatusCode {
	case 401:
		return fmt.Sprintf("%s (HTTP %d)", ErrAuthFailed.Error(), e.StatusCode)
	case 403:
		return fmt.Sprintf("%s (HTTP %d)", ErrRateLimit.Error(), e.StatusCode)
	case 429:
		return fmt.Sprintf("%s (HTTP %d)", ErrRateLimit.Error(), e.StatusCode)
	default:
		return fmt.Sprintf("GitHub API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
}
