package linear

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrNoAPIKey   = errors.New("LINEAR_API_KEY is not set; export LINEAR_API_KEY or add linear.api_key to .shipproof/config.yaml")
	ErrAuthFailed = errors.New("authentication failed; check your LINEAR_API_KEY")
	ErrRateLimit  = errors.New("Linear API rate limit exceeded; wait before retrying")
	ErrNotFound   = errors.New("resource not found in Linear")
)

func ResolveAPIKey(configDir string) (string, error) {
	if key := os.Getenv("LINEAR_API_KEY"); key != "" {
		return key, nil
	}
	return "", ErrNoAPIKey
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	switch e.StatusCode {
	case 401:
		return fmt.Sprintf("%s (HTTP %d)", ErrAuthFailed.Error(), e.StatusCode)
	case 429:
		return fmt.Sprintf("%s (HTTP %d)", ErrRateLimit.Error(), e.StatusCode)
	default:
		return fmt.Sprintf("Linear API error (HTTP %d): %s", e.StatusCode, e.Message)
	}
}
