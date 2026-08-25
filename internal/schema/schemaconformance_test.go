package schema

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// validate checks one decoded JSON value against one decoded JSON Schema.
//
// It supports the subset this repository's schema uses: type, required, enum,
// const, minLength, minimum, pattern, properties, items, and
// additionalProperties false. It reports every violation with its path. It is
// not a general JSON Schema implementation, and it adds no dependency.
func validate(schema map[string]any, value any, path string, problems *[]string) {
	fail := func(format string, arguments ...any) {
		*problems = append(*problems, path+": "+fmt.Sprintf(format, arguments...))
	}
	if want, ok := schema["const"]; ok && !equalJSON(want, value) {
		fail("value %v is not the constant %v", value, want)
	}
	if list, ok := schema["enum"].([]any); ok && !containsJSON(list, value) {
		fail("value %v is not in the enum %v", value, list)
	}
	if names := typeNames(schema["type"]); len(names) > 0 && !matchesAnyType(names, value) {
		fail("value %v is not of type %v", value, schema["type"])
	}
	switch typed := value.(type) {
	case string:
		if minimum, ok := schema["minLength"].(float64); ok && float64(len(typed)) < minimum {
			fail("string is shorter than %v", minimum)
		}
		if pattern, ok := schema["pattern"].(string); ok && !regexp.MustCompile(pattern).MatchString(typed) {
			fail("string %q does not match %q", typed, pattern)
		}
	case float64:
		if minimum, ok := schema["minimum"].(float64); ok && typed < minimum {
			fail("number %v is below %v", typed, minimum)
		}
	case []any:
		if items, ok := schema["items"].(map[string]any); ok {
			for index, item := range typed {
				validate(items, item, fmt.Sprintf("%s[%d]", path, index), problems)
			}
		}
	case map[string]any:
		properties, _ := schema["properties"].(map[string]any)
		for _, name := range stringList(schema["required"]) {
			if _, ok := typed[name]; !ok {
				fail("required property %q is missing", name)
			}
		}
		for _, name := range sortedKeys(typed) {
			child, ok := properties[name].(map[string]any)
			if !ok {
				if allow, ok := schema["additionalProperties"].(bool); ok && !allow {
					fail("property %q is not allowed", name)
				}
				continue
			}
			validate(child, typed[name], path+"."+name, problems)
		}
	}
}

// typeNames reads a type keyword that holds one name or a list of names.
func typeNames(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []any:
		return stringList(typed)
	}
	return nil
}

// matchesAnyType reports whether the value answers to one name of a union.
func matchesAnyType(names []string, value any) bool {
	for _, name := range names {
		if matchesType(name, value) {
			return true
		}
	}
	return false
}

func matchesType(want string, value any) bool {
	switch want {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == math.Trunc(number)
	}
	return true
}

func equalJSON(left, right any) bool {
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

func containsJSON(list []any, value any) bool {
	for _, item := range list {
		if equalJSON(item, value) {
			return true
		}
	}
	return false
}

func stringList(value any) []string {
	list, _ := value.([]any)
	names := make([]string, 0, len(list))
	for _, item := range list {
		if text, ok := item.(string); ok {
			names = append(names, text)
		}
	}
	return names
}

func sortedKeys(object map[string]any) []string {
	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func loadSchema(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "schemas", "v0.1", "evidence.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func checkPack(t *testing.T, name string, data []byte) {
	t.Helper()
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var problems []string
	validate(loadSchema(t), value, name, &problems)
	for _, problem := range problems {
		t.Errorf("%s", problem)
	}
}

func checkStruct(t *testing.T, name string, pack EvidencePack) {
	t.Helper()
	if err := pack.Validate(); err != nil {
		t.Fatalf("%s: Validate: %v", name, err)
	}
	data, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	checkPack(t, name, data)
}

func minimalPack() EvidencePack {
	return EvidencePack{
		SchemaVersion: CurrentVersion,
		ChangeID:      "SP-900",
		Intent:        IntentEvidence{SnapshotHash: "abc123"},
		Verification:  VerificationEvidence{Checks: []Check{}},
		Provenance:    PackProvenance{GeneratedAt: "2026-08-25T10:00:00Z", ShipProofVersion: CurrentVersion},
	}
}

// TestMinimalPackValidates proves the smallest pack the Go types produce
// answers to the schema.
func TestMinimalPackValidates(t *testing.T) {
	checkStruct(t, "minimal", minimalPack())
}

// TestFullPackValidates proves a pack that carries every optional block
// answers to the schema. Criterion S17 needs this.
func TestFullPackValidates(t *testing.T) {
	pack := minimalPack()
	pack.Intent = IntentEvidence{
		SnapshotHash:      "abc123",
		Requirements:      []Requirement{{ID: "SP-900-R1", VerificationRefs: []string{"go test ./..."}}},
		Stale:             true,
		CurrentSourceHash: "def456",
	}
	pack.Implementation = ImplementationEvidence{
		Commits:      []ImplementationCommit{{Hash: "1cceb33", Author: "Nate", Timestamp: "2026-08-25T09:00:00Z", Subject: "fix"}},
		ChangedFiles: []string{"internal/schema/evidence.go"},
		Additions:    10,
		Deletions:    2,
		DiffStat:     "1 file changed",
	}
	pack.Verification.Checks = []Check{
		{ID: "verification:run", Status: "pass", Source: "shipproof-runner", Provenance: ProvenanceObserved, Detail: "the gate passed"},
	}
	pack.AgentRun = &AgentRunMetadata{
		Provider: "anthropic", AgentVersion: "1.0", Model: "opus", StartedAt: "2026-08-25T08:00:00Z",
		EndedAt: "2026-08-25T09:00:00Z", SessionID: "s1", Cost: 1.25,
		Tokens: &TokenUsageMeta{Input: 100, Output: 200}, ToolCallCount: 12,
		ExitStatus: "completed", RawLogRef: ".shipproof/runs/SP-900/agent.log",
	}
	pack.AgentReview = &AgentReviewEvidence{
		Runner:   "claude",
		Findings: []AgentFinding{{Source: "reviewer", Summary: "one finding", Provenance: ProvenanceInferred}},
	}
	pack.Readiness = &ReadinessEvidence{ShapingRef: "shaping-one", BlockerCount: 2}
	pack.Review = &ReviewEvidence{
		Source: "github", PRNumber: 7, PRURL: "https://example.com/pr/7",
		OpenedAt: "2026-08-25T07:00:00Z", FirstReviewAt: "2026-08-25T07:30:00Z",
		ReviewCount: 1, CommentCount: 3, DistinctReviewers: 1,
		ReviewerLogins: []string{"nate"}, State: "open", CollectedAt: "2026-08-25T10:00:00Z",
	}
	pack.UnexplainedChange = &UnexplainedEvidence{
		CoverageAvailable:   true,
		LineFindings:        []UnexplainedLine{{File: "internal/a.go", Symbol: "func A()", StartLine: 10, EndLine: 12}},
		FileFindings:        []UnexplainedFile{{Path: "docs/workflow.md", IgnorePattern: "docs/**"}},
		UninstrumentedLines: 61,
	}
	checkStruct(t, "full", pack)
}

// TestEmptyUnexplainedFindingsValidate proves an unexplained-change section
// with no finding serialises as two empty arrays, never null, and still
// answers to the schema.
func TestEmptyUnexplainedFindingsValidate(t *testing.T) {
	pack := minimalPack()
	pack.UnexplainedChange = &UnexplainedEvidence{
		CoverageAvailable: false,
		LineFindings:      []UnexplainedLine{},
		FileFindings:      []UnexplainedFile{},
	}
	data, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !contains(body, `"line_findings":[]`) || !contains(body, `"file_findings":[]`) {
		t.Fatalf("an empty finding list did not serialise as an array: %s", body)
	}
	checkPack(t, "empty-findings", data)
}

func contains(haystack, needle string) bool {
	for index := 0; index+len(needle) <= len(haystack); index++ {
		if haystack[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestRecordedPacksValidate proves every evidence pack this repository holds
// on disk answers to the schema. Criterion S17 names a real pack.
func TestRecordedPacksValidate(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join("..", "..", ".shipproof", "changes", "*", "evidence-pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Skip("this repository holds no evidence pack")
	}
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			t.Fatal(err)
		}
		checkPack(t, filepath.ToSlash(match), data)
	}
}

// TestTheCheckerRejectsAnInvalidPack proves the checker reports a violation.
// A checker that passes everything proves nothing, so this test feeds it one
// pack that breaks four rules at once.
func TestTheCheckerRejectsAnInvalidPack(t *testing.T) {
	body := `{
	  "schema_version": "0.1",
	  "intent": {"snapshot_hash": "", "requirements": []},
	  "verification": {"checks": [{"id": "a", "status": "bogus", "source": "s", "provenance": "observed"}]},
	  "provenance": {"generated_at": "now", "shipproof_version": "0.1"},
	  "surprise": true
	}`
	var value any
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	var problems []string
	validate(loadSchema(t), value, "invalid", &problems)

	wanted := []string{
		`required property "change_id" is missing`,
		`property "surprise" is not allowed`,
		"is not in the enum",
		"string is shorter than 1",
	}
	for _, want := range wanted {
		found := false
		for _, problem := range problems {
			if contains(problem, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("the checker missed %q, problems = %v", want, problems)
		}
	}
}
