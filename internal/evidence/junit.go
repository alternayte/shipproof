package evidence

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/alternayte/shipproof/internal/schema"
)

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure"`
	Error     *junitError   `xml:"error"`
	Skipped   *junitSkipped `xml:"skipped"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
}

type junitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
}

type junitSkipped struct {
	Message string `xml:"message,attr"`
}

func ParseJUnit(path string) ([]schema.Check, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read junit file %s: %w", path, err)
	}

	var suite junitTestSuite
	if err := xml.Unmarshal(data, &suite); err == nil && len(suite.TestCases) > 0 {
		return testSuiteChecks(&suite), nil
	}

	var suites junitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		return nil, fmt.Errorf("parse junit xml %s: %w", path, err)
	}
	var checks []schema.Check
	for i := range suites.Suites {
		checks = append(checks, testSuiteChecks(&suites.Suites[i])...)
	}
	return checks, nil
}

func testSuiteChecks(suite *junitTestSuite) []schema.Check {
	var checks []schema.Check
	for _, tc := range suite.TestCases {
		id := tc.ClassName + "." + tc.Name

		var status string
		switch {
		case tc.Failure != nil:
			status = "fail"
		case tc.Error != nil:
			status = "fail"
		case tc.Skipped != nil:
			status = "skip"
		default:
			status = "pass"
		}

		checks = append(checks, schema.Check{
			ID:         id,
			Status:     status,
			Source:     "junit",
			Provenance: schema.ProvenanceObserved,
		})
	}
	return checks
}
