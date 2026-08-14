package evidence

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/shipproof/shipproof/internal/schema"
)

var ErrUnknownFormat = fmt.Errorf("unknown evidence format")

func ParseFiles(paths []string) ([]schema.Check, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	var allChecks []schema.Check
	for _, path := range paths {
		checks, err := parseFile(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		allChecks = append(allChecks, checks...)
	}
	return allChecks, nil
}

func parseFile(path string) ([]schema.Check, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "<") {
		return tryJUnit(path, data)
	}

	if strings.HasPrefix(trimmed, "{") {
		return trySARIF(path, data)
	}

	return nil, fmt.Errorf("%s: %w", path, ErrUnknownFormat)
}

func tryJUnit(path string, data []byte) ([]schema.Check, error) {
	var suite junitTestSuite
	if err := xml.Unmarshal(data, &suite); err == nil && len(suite.TestCases) > 0 {
		return testSuiteChecks(&suite), nil
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err == nil && len(suites.Suites) > 0 {
		var checks []schema.Check
		for i := range suites.Suites {
			checks = append(checks, testSuiteChecks(&suites.Suites[i])...)
		}
		return checks, nil
	}

	return nil, fmt.Errorf("%s: %w", path, ErrUnknownFormat)
}

func trySARIF(path string, data []byte) ([]schema.Check, error) {
	var log sarifLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("%s: %w", path, ErrUnknownFormat)
	}

	if log.Version == "" || len(log.Runs) == 0 {
		return nil, fmt.Errorf("%s: %w", path, ErrUnknownFormat)
	}

	return ParseSARIF(path)
}
