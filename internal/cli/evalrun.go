package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/shipproof/shipproof/internal/skills"
)

func runSkillEval(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof skill eval <check|list|show|record|results> ...")
		return 2
	}

	switch args[0] {
	case "record":
		return runSkillEvalRecord(args[1:], stdout, stderr)
	case "results":
		return runSkillEvalResults(args[1:], stdout, stderr)
	}

	manifest, err := skills.LoadBuiltInEvals()
	if err != nil {
		fmt.Fprintf(stderr, "eval manifest invalid: %v\n", err)
		return 1
	}
	switch args[0] {
	case "check":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "usage: shipproof skill eval check")
			return 2
		}
		fmt.Fprintf(stdout, "Skill eval manifest is valid: %d cases\n", len(manifest.Cases))
		return 0
	case "list":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "usage: shipproof skill eval list")
			return 2
		}
		for _, eval := range skills.SortedEvalCases(manifest) {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", eval.ID, eval.Skill, eval.Goal)
		}
		return 0
	case "show":
		if len(args) < 2 || len(args) > 3 {
			fmt.Fprintln(stderr, "usage: shipproof skill eval show <case-id> [--json]")
			return 2
		}
		eval, ok := skills.FindEvalCase(manifest, args[1])
		if !ok {
			fmt.Fprintf(stderr, "eval case %q not found\n", args[1])
			return 1
		}
		if len(args) == 3 {
			if args[2] != "--json" {
				fmt.Fprintf(stderr, "unknown option %q\n", args[2])
				return 2
			}
			encoder := json.NewEncoder(stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(eval); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			return 0
		}
		fmt.Fprintf(stdout, "%s (%s)\n%s\n", eval.ID, eval.Skill, eval.Goal)
		fmt.Fprintln(stdout, "Expected:")
		for _, item := range eval.Expected {
			fmt.Fprintf(stdout, "- %s\n", item)
		}
		if len(eval.Penalize) > 0 {
			fmt.Fprintln(stdout, "Penalize:")
			for _, item := range eval.Penalize {
				fmt.Fprintf(stdout, "- %s\n", item)
			}
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown skill eval command %q\n", args[0])
		return 2
	}
}

func runSkillEvalRecord(args []string, stdout, stderr io.Writer) int {
	caseID := ""
	condition := ""
	filePath := ""

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--condition":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--condition requires a value")
				return 2
			}
			condition = args[index+1]
			index++
		case "--file":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--file requires a path")
				return 2
			}
			filePath = args[index+1]
			index++
		default:
			if caseID != "" {
				fmt.Fprintln(stderr, "usage: shipproof skill eval record <case-id> --condition <without|previous|candidate> --file <result.json>")
				return 2
			}
			caseID = args[index]
		}
	}

	if caseID == "" || condition == "" || filePath == "" {
		fmt.Fprintln(stderr, "usage: shipproof skill eval record <case-id> --condition <without|previous|candidate> --file <result.json>")
		return 2
	}

	manifest, err := skills.LoadBuiltInEvals()
	if err != nil {
		fmt.Fprintf(stderr, "eval manifest invalid: %v\n", err)
		return 1
	}
	evalCase, ok := skills.FindEvalCase(manifest, caseID)
	if !ok {
		fmt.Fprintf(stderr, "eval case %q not found\n", caseID)
		return 1
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "read result file: %v\n", err)
		return 1
	}

	var run skills.EvalRun
	if err := json.Unmarshal(data, &run); err != nil {
		fmt.Fprintf(stderr, "parse result file: %v\n", err)
		return 1
	}
	if run.CaseID == "" {
		run.CaseID = caseID
	}
	if run.Skill == "" {
		run.Skill = evalCase.Skill
	}
	if run.Condition == "" {
		run.Condition = condition
	}
	if run.RecordedAt == "" {
		run.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	path, err := skills.RecordRun(root, run)
	if err != nil {
		fmt.Fprintf(stderr, "record eval run: %v\n", err)
		return 1
	}

	rel, _ := filepath.Rel(root, path)
	fmt.Fprintf(stdout, "Recorded eval run: %s\n", filepath.ToSlash(rel))
	return 0
}

func runSkillEvalResults(args []string, stdout, stderr io.Writer) int {
	regression := false
	caseID := ""
	for _, arg := range args {
		switch arg {
		case "--regression":
			regression = true
		default:
			if caseID != "" {
				fmt.Fprintln(stderr, "usage: shipproof skill eval results [case-id] [--regression]")
				return 2
			}
			caseID = arg
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	caseIDs := []string{}
	if caseID != "" {
		caseIDs = append(caseIDs, caseID)
	} else {
		dir := filepath.Join(root, "benchmarks", "skill-evals")
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(stdout, "No recorded eval runs.")
				return 0
			}
			fmt.Fprintf(stderr, "read skill-evals directory: %v\n", err)
			return 1
		}
		for _, entry := range entries {
			if entry.IsDir() {
				caseIDs = append(caseIDs, entry.Name())
			}
		}
	}

	total := 0
	regressions := 0
	for _, id := range caseIDs {
		runs, err := skills.LoadRuns(root, id)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", id, err)
			return 1
		}
		if len(runs) == 0 {
			fmt.Fprintf(stdout, "%s: no recorded runs\n", id)
			continue
		}
		for _, run := range runs {
			recall := "unknown"
			if run.BlockerRecall != nil {
				recall = fmt.Sprintf("%.2f", *run.BlockerRecall)
			}
			falseRate := "unknown"
			if run.FalseBlockerRate != nil {
				falseRate = fmt.Sprintf("%.2f", *run.FalseBlockerRate)
			}
			fmt.Fprintf(stdout, "%s\t%s\tsuccess=%t\trecall=%s\tfalse-blockers=%s\tquestions=%d\n",
				id, run.Condition, run.TaskSuccess, recall, falseRate, run.QuestionsAsked)
			total++
		}
		if regression {
			isRegression, reason := skills.Regression(runs)
			if isRegression {
				fmt.Fprintf(stdout, "%s: REGRESSION — %s\n", id, reason)
				regressions++
			}
		}
	}

	fmt.Fprintf(stdout, "%d runs across %d cases\n", total, len(caseIDs))
	if regression && regressions > 0 {
		return 1
	}
	return 0
}
