package evidence

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alternayte/shipproof/internal/schema"
)

type sarifLog struct {
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name string `json:"name"`
}

type sarifResult struct {
	RuleID  string       `json:"ruleId,omitempty"`
	Level   string       `json:"level"`
	Message sarifMessage `json:"message"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

func ParseSARIF(path string) ([]schema.Check, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sarif file %s: %w", path, err)
	}

	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("parse sarif json %s: %w", path, err)
	}

	if log.Version != "2.1.0" {
		return nil, fmt.Errorf("parse sarif %s: unsupported version %q, expected 2.1.0", path, log.Version)
	}

	var checks []schema.Check
	for _, run := range log.Runs {
		for i, result := range run.Results {
			id := result.RuleID
			if id == "" {
				id = fmt.Sprintf("result-%d", i)
			}

			status := mapSARIFLevel(result.Level)

			checks = append(checks, schema.Check{
				ID:         id,
				Status:     status,
				Source:     "sarif",
				Provenance: schema.ProvenanceObserved,
			})
		}
	}
	return checks, nil
}

func mapSARIFLevel(level string) string {
	switch level {
	case "error":
		return "fail"
	case "warning":
		return "unknown"
	case "note":
		return "unknown"
	case "none":
		return "skip"
	default:
		return "unknown"
	}
}
