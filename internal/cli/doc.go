package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shipproof/shipproof/internal/document"
	"github.com/shipproof/shipproof/internal/language"
)

func runDoc(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: shipproof doc <status|review> <file> [--kind prd|sdd] [--json]")
		return 2
	}

	action := args[0]
	if action != "status" && action != "review" {
		fmt.Fprintf(stderr, "unknown doc command %q\n", action)
		return 2
	}

	path := args[1]
	kind, jsonOutput, err := parseDocOptions(path, args[2:])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	root, err := findRepositoryRoot(path)
	if err != nil {
		root, _ = os.Getwd()
	}
	glossary, err := language.LoadGlossary(filepath.Join(root, ".shipproof", "glossary.yaml"))
	if err != nil {
		fmt.Fprintf(stderr, "load glossary: %v\n", err)
		return 1
	}

	review, err := document.AnalyzeFile(path, document.AnalyzeOptions{
		Kind: kind, Glossary: glossary, Language: language.DefaultOptions(),
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(review); err != nil {
			fmt.Fprintf(stderr, "write review: %v\n", err)
			return 1
		}
		return 0
	}

	printDocReview(stdout, review, action == "review")
	return 0
}

func parseDocOptions(path string, args []string) (document.Kind, bool, error) {
	kind := inferDocumentKind(path)
	jsonOutput := false
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--json":
			jsonOutput = true
		case "--kind":
			if index+1 >= len(args) {
				return "", false, fmt.Errorf("--kind requires prd or sdd")
			}
			parsed, err := document.ParseKind(strings.ToLower(args[index+1]))
			if err != nil {
				return "", false, err
			}
			kind = parsed
			index++
		default:
			return "", false, fmt.Errorf("unknown option %q", args[index])
		}
	}
	if kind == "" {
		return "", false, fmt.Errorf("cannot infer document kind; use --kind prd or --kind sdd")
	}
	return kind, jsonOutput, nil
}

func inferDocumentKind(path string) document.Kind {
	value := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(value, "/prd/"), strings.Contains(filepath.Base(value), "prd"):
		return document.KindPRD
	case strings.Contains(value, "/sdd/"), strings.Contains(value, "/design/"), strings.Contains(filepath.Base(value), "sdd"):
		return document.KindSDD
	default:
		return ""
	}
}

func findRepositoryRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, ".shipproof")); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("ShipProof repository root not found")
		}
		abs = parent
	}
}

func printDocReview(w io.Writer, review document.Review, includeFindings bool) {
	provisional := ""
	if review.Readiness.Provisional {
		provisional = " (provisional: semantic review pending)"
	}
	fmt.Fprintf(w, "%s: %s%s\n", strings.ToUpper(string(review.Kind)), strings.ToUpper(string(review.Readiness.State)), provisional)
	fmt.Fprintf(w, "Blockers %d | Decisions %d | Assumptions %d | Risks %d | Suggestions %d | Nits %d\n",
		review.Readiness.Blockers, review.Readiness.Decisions, review.Readiness.Assumptions,
		review.Readiness.Risks, review.Readiness.Suggestions, review.Readiness.Nits)

	if !includeFindings || len(review.Findings) == 0 {
		return
	}
	fmt.Fprintln(w)
	for _, finding := range review.Findings {
		location := ""
		if finding.Line > 0 {
			location = fmt.Sprintf(" line %d", finding.Line)
		}
		fmt.Fprintf(w, "[%s] %s%s — %s\n", strings.ToUpper(string(finding.Class)), finding.ID, location, finding.Claim)
		fmt.Fprintf(w, "  %s\n", finding.Reason)
		if finding.SuggestedAction != "" {
			fmt.Fprintf(w, "  Action: %s\n", finding.SuggestedAction)
		}
	}
}
