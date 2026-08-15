package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EvalCondition names one comparison arm from SDD §11.
type EvalCondition string

const (
	ConditionWithout   EvalCondition = "without"
	ConditionPrevious  EvalCondition = "previous"
	ConditionCandidate EvalCondition = "candidate"
)

// EvalRun records one measured eval execution for one case.
// Fields are optional unless stated. Unknown fields stay absent.
type EvalRun struct {
	SchemaVersion      string   `json:"schema_version"`
	CaseID             string   `json:"case_id"`
	Skill              string   `json:"skill"`
	Condition          string   `json:"condition"`
	RecordedAt         string   `json:"recorded_at"`
	TaskSuccess        bool     `json:"task_success"`
	BlockerRecall      *float64 `json:"blocker_recall,omitempty"`
	FalseBlockerRate   *float64 `json:"false_blocker_rate,omitempty"`
	QuestionsAsked     int      `json:"questions_asked"`
	HumanIntervention  bool     `json:"human_intervention"`
	ElapsedSeconds     *float64 `json:"elapsed_seconds,omitempty"`
	TokenUsageInput    *int64   `json:"token_usage_input,omitempty"`
	TokenUsageOutput   *int64   `json:"token_usage_output,omitempty"`
	ObservedCost       *float64 `json:"observed_cost,omitempty"`
	VerificationStatus *string  `json:"verification_status,omitempty"`
	HumanRating        *int     `json:"human_rating,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

func (run EvalRun) Validate() error {
	if run.SchemaVersion != "0.1" {
		return fmt.Errorf("schema_version must be %q", "0.1")
	}
	if strings.TrimSpace(run.CaseID) == "" {
		return errors.New("case_id is required")
	}
	if strings.TrimSpace(run.Skill) == "" {
		return errors.New("skill is required")
	}
	switch EvalCondition(run.Condition) {
	case ConditionWithout, ConditionPrevious, ConditionCandidate:
	default:
		return fmt.Errorf("condition must be without, previous, or candidate; got %q", run.Condition)
	}
	if _, err := time.Parse(time.RFC3339, run.RecordedAt); err != nil {
		return errors.New("recorded_at must be RFC 3339")
	}
	if run.QuestionsAsked < 0 {
		return errors.New("questions_asked cannot be negative")
	}
	for name, rate := range map[string]*float64{
		"blocker_recall":     run.BlockerRecall,
		"false_blocker_rate": run.FalseBlockerRate,
	} {
		if rate != nil && (*rate < 0 || *rate > 1) {
			return fmt.Errorf("%s must be between 0 and 1", name)
		}
	}
	return nil
}

// EvalRunDir returns the storage directory for one case and condition.
func EvalRunDir(root, caseID string, condition EvalCondition) string {
	return filepath.Join(root, "benchmarks", "skill-evals", caseID, string(condition))
}

// NextRunPath returns the next unused run path for a case and condition.
func NextRunPath(root, caseID string, condition EvalCondition) string {
	dir := EvalRunDir(root, caseID, condition)
	index := 1
	for {
		candidate := filepath.Join(dir, fmt.Sprintf("run-%d.json", index))
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
		index++
	}
}

// RecordRun validates and writes an eval run. The run index is assigned
// automatically.
func RecordRun(root string, run EvalRun) (string, error) {
	if err := run.Validate(); err != nil {
		return "", err
	}

	path := NextRunPath(root, run.CaseID, EvalCondition(run.Condition))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create eval run directory: %w", err)
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode eval run: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write eval run: %w", err)
	}

	return path, nil
}

// LoadRuns reads every recorded run for one case across conditions.
func LoadRuns(root, caseID string) ([]EvalRun, error) {
	dir := filepath.Join(root, "benchmarks", "skill-evals", caseID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read eval directory: %w", err)
	}

	var runs []EvalRun
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		conditionDir := filepath.Join(dir, entry.Name())
		files, err := os.ReadDir(conditionDir)
		if err != nil {
			return nil, fmt.Errorf("read condition directory: %w", err)
		}
		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(conditionDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("read eval run: %w", err)
			}
			var run EvalRun
			if err := json.Unmarshal(data, &run); err != nil {
				return nil, fmt.Errorf("parse eval run %s: %w", file.Name(), err)
			}
			if err := run.Validate(); err != nil {
				return nil, fmt.Errorf("validate eval run %s: %w", file.Name(), err)
			}
			runs = append(runs, run)
		}
	}
	return runs, nil
}

// Regression compares the latest candidate run against the latest baseline
// run. The baseline is the previous condition when present, otherwise the
// without condition. It reports a regression when the candidate fails a task
// the baseline passed, recalls fewer blockers, or raises the false blocker
// rate.
func Regression(runs []EvalRun) (bool, string) {
	var candidate, baseline *EvalRun
	for index := range runs {
		run := &runs[index]
		switch EvalCondition(run.Condition) {
		case ConditionCandidate:
			candidate = run
		case ConditionPrevious:
			baseline = run
		}
	}
	if baseline == nil {
		for index := range runs {
			if EvalCondition(runs[index].Condition) == ConditionWithout {
				baseline = &runs[index]
			}
		}
	}
	if candidate == nil || baseline == nil {
		return false, ""
	}

	switch {
	case baseline.TaskSuccess && !candidate.TaskSuccess:
		return true, "task success regressed"
	case baseline.BlockerRecall != nil && candidate.BlockerRecall != nil &&
		*candidate.BlockerRecall < *baseline.BlockerRecall:
		return true, "blocker recall decreased"
	case baseline.FalseBlockerRate != nil && candidate.FalseBlockerRate != nil &&
		*candidate.FalseBlockerRate > *baseline.FalseBlockerRate:
		return true, "false blocker rate increased"
	default:
		return false, ""
	}
}
