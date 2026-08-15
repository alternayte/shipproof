package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	skillassets "github.com/alternayte/shipproof/skills"
)

type EvalCase struct {
	ID       string   `json:"id"`
	Skill    string   `json:"skill"`
	Goal     string   `json:"goal"`
	Prompt   string   `json:"prompt,omitempty"`
	Context  []string `json:"context,omitempty"`
	Expected []string `json:"expected"`
	Penalize []string `json:"penalize"`
}

type EvalManifest struct {
	SchemaVersion string     `json:"schema_version"`
	Cases         []EvalCase `json:"cases"`
}

func LoadBuiltInEvals() (EvalManifest, error) {
	data, err := skillassets.Catalog.ReadFile("evals/spec-skills.cases.json")
	if err != nil {
		return EvalManifest{}, fmt.Errorf("read built-in evals: %w", err)
	}
	var manifest EvalManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return EvalManifest{}, fmt.Errorf("parse built-in evals: %w", err)
	}
	if err := ValidateEvalManifest(manifest); err != nil {
		return EvalManifest{}, err
	}
	return manifest, nil
}

func ValidateEvalManifest(manifest EvalManifest) error {
	if manifest.SchemaVersion != "0.1" {
		return fmt.Errorf("eval schema_version must be %q", "0.1")
	}
	if len(manifest.Cases) == 0 {
		return errors.New("eval manifest has no cases")
	}
	seen := map[string]struct{}{}
	for index, eval := range manifest.Cases {
		if strings.TrimSpace(eval.ID) == "" {
			return fmt.Errorf("case %d id is required", index+1)
		}
		if _, exists := seen[eval.ID]; exists {
			return fmt.Errorf("duplicate eval case id %q", eval.ID)
		}
		seen[eval.ID] = struct{}{}
		if strings.TrimSpace(eval.Skill) == "" {
			return fmt.Errorf("case %q skill is required", eval.ID)
		}
		if !namePattern.MatchString(eval.Skill) {
			return fmt.Errorf("case %q has invalid skill name %q", eval.ID, eval.Skill)
		}
		if strings.TrimSpace(eval.Goal) == "" {
			return fmt.Errorf("case %q goal is required", eval.ID)
		}
		if len(eval.Expected) == 0 {
			return fmt.Errorf("case %q needs at least one expected behavior", eval.ID)
		}
	}
	return nil
}

func FindEvalCase(manifest EvalManifest, id string) (EvalCase, bool) {
	for _, eval := range manifest.Cases {
		if eval.ID == id {
			return eval, true
		}
	}
	return EvalCase{}, false
}

func SortedEvalCases(manifest EvalManifest) []EvalCase {
	result := append([]EvalCase(nil), manifest.Cases...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
