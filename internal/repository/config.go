package repository

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CaptureLevel is one evidence capture profile from SDD §18.
type CaptureLevel string

const (
	CaptureMetadata CaptureLevel = "metadata"
	CaptureRedacted CaptureLevel = "redacted"
	CaptureFull     CaptureLevel = "full"
)

var (
	ErrCommandMissing = errors.New("verification.command is not set in config")
	ErrConfigNotFound = errors.New("config file not found")
)

// Config is the parsed .shipproof/config.yaml content.
type Config struct {
	Verification VerificationConfig
	Evidence     EvidenceConfig
}

type VerificationConfig struct {
	Command  string
	Coverage CoverageConfig
	// UnexplainedIgnore holds glob patterns. A changed file that matches a
	// pattern appears in the report with its reason and counts toward no
	// total.
	UnexplainedIgnore []string
}

type CoverageConfig struct {
	// Command is a template. {{profile}} takes the output profile path and
	// {{target}} takes the proof target.
	Command string
	// Format names the profile dialect. Only "go" is supported.
	Format string
}

type EvidenceConfig struct {
	// Capture defaults to metadata. Redacted and full store transcripts
	// under .shipproof/runs/<change-id>/agent-raw/.
	Capture CaptureLevel
}

func LoadConfig(root string) (Config, error) {
	cfg, err := parseConfigFile(root)
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(cfg.Verification.Command) == "" {
		return Config{}, ErrCommandMissing
	}
	return cfg, nil
}

// LoadEvidenceConfig reads the evidence section only. A missing config file
// defaults to metadata capture. The verification command is not required.
func LoadEvidenceConfig(root string) (Config, error) {
	cfg, err := parseConfigFile(root)
	if errors.Is(err, ErrConfigNotFound) {
		return Config{Evidence: EvidenceConfig{Capture: CaptureMetadata}}, nil
	}
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parseConfigFile(root string) (Config, error) {
	configPath := filepath.Join(root, ".shipproof", "config.yaml")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("%w: run shipproof init first", ErrConfigNotFound)
		}
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	return parseConfig(file)
}

// parseConfig reads the small YAML subset that ShipProof writes. It supports
// one top-level section, one nested block inside a section, and a list of
// scalar items under a key.
func parseConfig(file *os.File) (Config, error) {
	cfg := Config{Evidence: EvidenceConfig{Capture: CaptureMetadata}}

	scanner := bufio.NewScanner(file)
	section := ""
	block := ""
	blockIndent := -1
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := indentWidth(line)
		if indent == 0 {
			block, blockIndent = "", -1
			if strings.HasSuffix(trimmed, ":") {
				section = strings.TrimSuffix(trimmed, ":")
				continue
			}
		}
		if blockIndent >= 0 && indent <= blockIndent {
			block, blockIndent = "", -1
		}

		if item, ok := strings.CutPrefix(trimmed, "- "); ok {
			if section == "verification" && block == "unexplained_ignore" {
				cfg.Verification.UnexplainedIgnore = append(
					cfg.Verification.UnexplainedIgnore, strings.Trim(strings.TrimSpace(item), `"'`))
			}
			continue
		}

		key, value, ok := splitKV(trimmed)
		if !ok {
			continue
		}
		if value == "" {
			block, blockIndent = key, indent
			continue
		}

		assign(&cfg, section, block, key, value)
	}

	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	switch cfg.Evidence.Capture {
	case CaptureMetadata, CaptureRedacted, CaptureFull:
	default:
		return Config{}, fmt.Errorf("evidence.capture must be metadata, redacted, or full; got %q", cfg.Evidence.Capture)
	}

	return cfg, nil
}

func assign(cfg *Config, section, block, key, value string) {
	switch section {
	case "verification":
		switch block {
		case "":
			if key == "command" {
				cfg.Verification.Command = value
			}
		case "coverage":
			switch key {
			case "command":
				cfg.Verification.Coverage.Command = value
			case "format":
				cfg.Verification.Coverage.Format = value
			}
		}
	case "evidence":
		if block == "" && key == "capture" {
			cfg.Evidence.Capture = CaptureLevel(value)
		}
	}
}

// indentWidth counts the leading whitespace of one line. A tab counts as one.
func indentWidth(line string) int {
	width := 0
	for _, character := range line {
		if character != ' ' && character != '\t' {
			break
		}
		width++
	}
	return width
}

func splitKV(line string) (string, string, bool) {
	index := strings.Index(line, ":")
	if index < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:index])
	value := strings.TrimSpace(line[index+1:])
	value = strings.Trim(value, `"'`)
	return key, value, key != ""
}
