package verify

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Command string
}

var ErrCommandMissing = errors.New("verification.command is not set in config")

func LoadConfig(root string) (Config, error) {
	configPath := filepath.Join(root, ".shipproof", "config.yaml")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config file not found: run shipproof init first")
		}
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	command, err := parseVerificationCommand(file)
	if err != nil {
		return Config{}, err
	}

	return Config{Command: command}, nil
}

func parseVerificationCommand(file *os.File) (string, error) {
	scanner := bufio.NewScanner(file)
	inVerification := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if trimmed == "verification:" {
			inVerification = true
			continue
		}

		if inVerification {
			if strings.HasPrefix(trimmed, "command:") {
				value := strings.TrimSpace(strings.TrimPrefix(trimmed, "command:"))
				value = strings.Trim(value, `"'`)
				if value == "" {
					return "", ErrCommandMissing
				}
				return value, nil
			}
			if !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "\t") {
				inVerification = false
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}
	return "", ErrCommandMissing
}
