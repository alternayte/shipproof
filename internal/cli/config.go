package cli

import (
	"fmt"
	"io"

	"github.com/alternayte/shipproof/internal/repository"
)

// runConfig implements `shipproof config get|set`.
func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: shipproof config <get|set> ...")
		return 2
	}
	switch args[0] {
	case "get":
		return runConfigGet(args[1:], stdout, stderr)
	case "set":
		return runConfigSet(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown config command %q\n", args[0])
		return 2
	}
}

func runConfigGet(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: shipproof config get <key>")
		return 2
	}
	root, err := findRepositoryRoot(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	value, scope, err := repository.GetValue(root, args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "%s\n", value)
	fmt.Fprintf(stderr, "scope: %s\n", scope)
	return 0
}

func runConfigSet(args []string, stdout, stderr io.Writer) int {
	scope := repository.ScopeLocal
	var positional []string
	for _, arg := range args {
		switch arg {
		case "--global":
			scope = repository.ScopeGlobal
		case "--local":
			scope = repository.ScopeLocal
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) != 2 {
		fmt.Fprintln(stderr, "usage: shipproof config set <key> <value> [--global|--local]")
		return 2
	}

	root, err := findRepositoryRoot(".")
	if err != nil && scope == repository.ScopeLocal {
		fmt.Fprintln(stderr, err)
		return 1
	}

	path, err := repository.SetValue(root, positional[0], positional[1], scope)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Set %s in %s scope\n", positional[0], scope)
	fmt.Fprintf(stderr, "file: %s\n", path)
	return 0
}
