package cli

import (
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
