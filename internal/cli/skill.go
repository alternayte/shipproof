package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/shipproof/shipproof/internal/skills"
)

func runSkill(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof skill <check|eval> ...")
		return 2
	}
	if args[0] == "eval" {
		return runSkillEval(args[1:], stdout, stderr)
	}
	if args[0] != "check" {
		fmt.Fprintln(stderr, "usage: shipproof skill <check|eval> ...")
		return 2
	}
	path := "skills"
	if len(args) > 2 {
		fmt.Fprintln(stderr, "usage: shipproof skill check [catalog-directory]")
		return 2
	}
	if len(args) == 2 {
		path = args[1]
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(stderr, "resolve skill catalog: %v\n", err)
		return 1
	}
	if err := skills.ValidateCatalog(abs); err != nil {
		fmt.Fprintf(stderr, "skill validation failed:\n%v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Skill catalog is valid: %s\n", abs)
	return 0
}

func runSkillEval(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof skill eval <check|list|show> ...")
		return 2
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
