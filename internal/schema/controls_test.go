package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// fieldPattern reads a backtick-quoted pack field from the document. A field
// reference holds a dot or a bracket, so a plain word never matches.
var fieldPattern = regexp.MustCompile("`([a-z_]+(?:\\[\\])?(?:\\.[a-z_0-9]+)+)`")

func readControls(t *testing.T) string {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	path := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "docs", "controls.md"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read docs/controls.md: %v", err)
	}
	return string(data)
}

// packFields walks the Go types and reports every JSON path that a pack can
// hold. A path through a slice carries the `[]` marker, which matches the way
// the document writes it.
func packFields() map[string]bool {
	found := map[string]bool{}
	walk(reflect.TypeOf(EvidencePack{}), "", found)
	return found
}

func walk(target reflect.Type, prefix string, found map[string]bool) {
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if target.Kind() == reflect.Slice {
		walk(target.Elem(), prefix+"[]", found)
		return
	}
	if target.Kind() != reflect.Struct {
		return
	}
	for index := 0; index < target.NumField(); index++ {
		field := target.Field(index)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		path := tag
		if prefix != "" {
			path = prefix + "." + tag
		}
		found[path] = true
		walk(field.Type, path, found)
	}
}

// TestEveryFieldInTheControlsDocumentExists is the proof for row P4 of the
// definition of done.
func TestEveryFieldInTheControlsDocumentExists(t *testing.T) {
	document := readControls(t)
	fields := packFields()

	matches := fieldPattern.FindAllStringSubmatch(document, -1)
	if len(matches) == 0 {
		t.Fatal("docs/controls.md names no pack field")
	}
	for _, match := range matches {
		if !fields[match[1]] {
			t.Errorf("docs/controls.md names the field %q, which the schema does not hold", match[1])
		}
	}
}

// TestTheControlsDocumentAnswersEveryAuditQuestion holds requirement R3.
func TestTheControlsDocumentAnswersEveryAuditQuestion(t *testing.T) {
	document := readControls(t)
	for _, question := range []string{
		"Was the approved requirement the one that shipped?",
		"Was each requirement tested before release?",
		"Is the result machine-observed or asserted?",
		"Did the change stay inside the approved scope?",
		"Which model produced the change, and under which session?",
		"Can the record be altered after the fact?",
	} {
		if !strings.Contains(document, question) {
			t.Errorf("docs/controls.md never answers %q", question)
		}
	}
}

// TestTheControlsDocumentNamesTheThreeFrameworks holds requirement R4.
func TestTheControlsDocumentNamesTheThreeFrameworks(t *testing.T) {
	document := readControls(t)
	for _, framework := range []string{"SOC 2", "EU AI Act", "SOX"} {
		if !strings.Contains(document, framework) {
			t.Errorf("docs/controls.md never names %q", framework)
		}
	}
}

// TestTheControlsDocumentClaimsNoCertification holds requirement R5. Section
// 11 says to state the mapping as a claim about evidence, never as a
// certification.
func TestTheControlsDocumentClaimsNoCertification(t *testing.T) {
	document := readControls(t)
	lower := strings.ToLower(document)
	for _, claim := range []string{
		"shipproof certifies",
		"is soc 2 compliant",
		"is compliant with",
		"guarantees compliance",
		"this is a certification",
	} {
		if strings.Contains(lower, claim) {
			t.Errorf("docs/controls.md holds the certification claim %q", claim)
		}
	}
	if !strings.Contains(document, "ShipProof issues no certification.") {
		t.Error("docs/controls.md never states that ShipProof issues no certification")
	}
}

// TestTheSchemaDocumentHoldsEveryNamedField binds the published JSON Schema to
// the same field list, so a reader of the schema sees what the document names.
func TestTheSchemaDocumentHoldsEveryNamedField(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	path := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..",
		"schemas", "v"+CurrentVersion, "evidence.schema.json"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	for _, match := range fieldPattern.FindAllStringSubmatch(readControls(t), -1) {
		if !schemaHolds(document, match[1]) {
			t.Errorf("schemas/v%s does not describe the field %q", CurrentVersion, match[1])
		}
	}
}

// schemaHolds walks one dotted path through a JSON Schema document.
func schemaHolds(node map[string]any, path string) bool {
	current := node
	for _, part := range strings.Split(path, ".") {
		name := strings.TrimSuffix(part, "[]")
		properties, ok := current["properties"].(map[string]any)
		if !ok {
			return false
		}
		child, ok := properties[name].(map[string]any)
		if !ok {
			return false
		}
		if items, ok := child["items"].(map[string]any); ok {
			child = items
		}
		current = child
	}
	return true
}
