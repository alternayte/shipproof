package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shipproof/shipproof/internal/shaping"
)

func runShape(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof shape <prd|sdd|issue|start|status|check> ...")
		return 2
	}

	switch args[0] {
	case "prd", "sdd", "issue":
		return runShapeStart(append([]string{args[0]}, args[1:]...), stdout, stderr)
	case "start":
		return runShapeStart(args[1:], stdout, stderr)
	case "status":
		return runShapeStatus(args[1:], stdout, stderr)
	case "check":
		return runShapeCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown shape command %q\n", args[0])
		return 2
	}
}

func runShapeStart(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: shipproof shape start <prd|sdd|issue> <subject> [--id id] [--source path]")
		return 2
	}
	kind := strings.ToLower(args[0])
	subject := args[1]
	if kind != "prd" && kind != "sdd" && kind != "issue" {
		fmt.Fprintln(stderr, "document kind must be prd, sdd, or issue")
		return 2
	}

	var id, source string
	for index := 2; index < len(args); index++ {
		switch args[index] {
		case "--id":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--id requires a value")
				return 2
			}
			id = args[index+1]
			index++
		case "--source":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "--source requires a path")
				return 2
			}
			source = args[index+1]
			index++
		default:
			fmt.Fprintf(stderr, "unknown option %q\n", args[index])
			return 2
		}
	}

	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, "ShipProof repository root not found; run shipproof init first")
		return 1
	}
	if source != "" {
		if abs, err := filepath.Abs(source); err == nil {
			if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
				source = filepath.ToSlash(rel)
			}
		}
	}

	session, path, err := shaping.Start(root, shaping.StartOptions{Kind: kind, Subject: subject, ID: id, Source: source})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	rel, _ := filepath.Rel(root, path)
	fmt.Fprintf(stdout, "Started %s shaping session %q\n", strings.ToUpper(kind), session.SessionID)
	fmt.Fprintf(stdout, "State: %s\n", strings.ToUpper(string(session.State)))
	fmt.Fprintf(stdout, "File: %s\n", filepath.ToSlash(rel))
	fmt.Fprintln(stdout, "Next: use the matching ShipProof shaping skill. The skill should update this session and stop when the readiness gate is satisfied.")
	return 0
}

func runShapeStatus(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintln(stderr, "usage: shipproof shape status <id> [--json]")
		return 2
	}
	jsonOutput := false
	if len(args) == 2 {
		if args[1] != "--json" {
			fmt.Fprintf(stderr, "unknown option %q\n", args[1])
			return 2
		}
		jsonOutput = true
	}
	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	session, path, err := shaping.Load(root, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(session); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}

	rel, _ := filepath.Rel(root, path)
	fmt.Fprintf(stdout, "%s — %s\n", session.Subject, strings.ToUpper(string(session.State)))
	fmt.Fprintf(stdout, "Kind: %s | Decisions: %d | Assumptions: %d | Risks: %d | Unknowns: %d\n",
		strings.ToUpper(session.DocumentKind), len(session.Decisions), len(session.Assumptions), len(session.Risks), len(session.Unknowns))
	fmt.Fprintf(stdout, "Readiness blockers: %d | Decisions required: %d\n", len(session.Readiness.Blockers), len(session.Readiness.DecisionsRequired))
	fmt.Fprintf(stdout, "File: %s\n", filepath.ToSlash(rel))
	return 0
}

func runShapeCheck(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof shape check <id-or-file>")
		return 2
	}
	path := args[0]
	if _, err := os.Stat(path); err != nil {
		root, rootErr := findRepositoryRoot(".")
		if rootErr != nil {
			fmt.Fprintln(stderr, rootErr)
			return 1
		}
		path = shaping.Path(root, args[0])
	}
	session, err := shaping.CheckFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "invalid shaping session: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Shaping session is valid: %s (%s)\n", session.Subject, session.State)
	return 0
}
