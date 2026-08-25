package verify

import (
	"errors"
	"fmt"

	"github.com/alternayte/shipproof/internal/repository"
)

// Config holds the repository gate command. The gate is the authority on
// whether the repository passes.
type Config struct {
	Command string
}

var ErrCommandMissing = errors.New("verification.command is not set in config")

// LoadConfig reads the gate command through the single configuration parser.
func LoadConfig(root string) (Config, error) {
	cfg, err := repository.LoadConfig(root)
	if errors.Is(err, repository.ErrConfigNotFound) {
		return Config{}, fmt.Errorf("config file not found: run shipproof init first")
	}
	if errors.Is(err, repository.ErrCommandMissing) {
		return Config{}, ErrCommandMissing
	}
	if err != nil {
		return Config{}, err
	}
	return Config{Command: cfg.Verification.Command}, nil
}
