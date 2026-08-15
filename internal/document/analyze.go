package document

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/alternayte/shipproof/internal/language"
)

type AnalyzeOptions struct {
	Kind     Kind
	Glossary language.Glossary
	Language language.Options
}

type parsedDocument struct {
	lines    []string
	headings []heading
	plain    string
}

type heading struct {
	line  int
	level int
	text  string
}

var headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
var placeholderPattern = regexp.MustCompile(`(?i)\b(TODO|TBD|FIXME|XXX|fill this|to be decided)\b`)
var genericNFRPattern = regexp.MustCompile(`(?i)\b(must|should|shall|needs? to|will be)\s+(be\s+)?(highly\s+)?(scalable|secure|maintainable|reliable|available|performant|fast|resilient)\b`)

func AnalyzeFile(path string, options AnalyzeOptions) (Review, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Review{}, fmt.Errorf("read document: %w", err)
	}
	return Analyze(path, string(data), options), nil
}

func Analyze(path, content string, options AnalyzeOptions) Review {
	doc := parse(content)
	findings := make([]Finding, 0)
	findings = append(findings, structuralFindings(doc, options.Kind)...)
	findings = append(findings, placeholderFindings(doc)...)
	findings = append(findings, nfrFindings(doc)...)
	findings = append(findings, languageFindings(content, options.Glossary, options.Language)...)

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Line == findings[j].Line {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].Line < findings[j].Line
	})

	return Review{
		Kind:      options.Kind,
		Path:      filepath.Clean(path),
		Findings:  findings,
		Readiness: AssessReadiness(findings, true),
	}
}

func parse(content string) parsedDocument {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	var headings []heading
	var plain strings.Builder
	inFence := false

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if match := headingPattern.FindStringSubmatch(line); match != nil {
			headings = append(headings, heading{line: len(lines), level: len(match[1]), text: strings.TrimSpace(match[2])})
		}
		plain.WriteString(line)
		plain.WriteByte('\n')
	}

	return parsedDocument{lines: lines, headings: headings, plain: plain.String()}
}

func structuralFindings(doc parsedDocument, kind Kind) []Finding {
	var findings []Finding
	plain := strings.ToLower(doc.plain)

	checks := []struct {
		id       string
		category string
		terms    []string
		class    FindingClass
		claim    string
		reason   string
		action   string
	}{
		{"DOC-PROBLEM", "intent", []string{"problem", "pain", "need", "opportunity"}, FindingBlocker, "The document does not make the problem explicit.", "A buildable specification needs a problem or need that can be distinguished from the solution.", "State the problem, observed pain, or explicit hypothesis."},
		{"DOC-OUTCOME", "intent", []string{"outcome", "goal", "success", "result"}, FindingBlocker, "The desired outcome is not explicit.", "The next stage needs to know what successful behavior or result means.", "State an observable desired outcome."},
		{"DOC-SCOPE", "scope", []string{"scope", "in scope", "non-goal", "out of scope", "no-go", "boundary"}, FindingSuggestion, "Scope boundaries are not explicit.", "Explicit boundaries reduce accidental expansion, especially during agent-assisted implementation.", "Add only the boundaries that materially constrain the work."},
		{"DOC-UNKNOWN", "uncertainty", []string{"unknown", "open question", "assumption", "risk"}, FindingSuggestion, "The document does not expose assumptions, risks, or unknowns.", "Unknown information should remain visible rather than become plausible generated detail.", "Record material assumptions, risks, or open questions. Omit the section when none are material."},
	}

	if kind == KindPRD {
		checks = append(checks,
			struct {
				id       string
				category string
				terms    []string
				class    FindingClass
				claim    string
				reason   string
				action   string
			}{"PRD-ACTOR", "users", []string{"user", "customer", "actor", "persona", "operator", "admin"}, FindingBlocker, "The affected user or actor is not explicit.", "Product intent cannot be evaluated without knowing who experiences the problem or behavior.", "Name the relevant user or actor."},
			struct {
				id       string
				category string
				terms    []string
				class    FindingClass
				claim    string
				reason   string
				action   string
			}{"PRD-ACCEPT", "acceptance", []string{"acceptance", "verify", "measur", "success criteria", "expected behavior"}, FindingSuggestion, "Acceptance is not explicit.", "Important behavior should be testable or inspectable before implementation starts.", "Describe how the important outcomes will be evaluated."},
		)
	}

	if kind == KindSDD {
		checks = append(checks,
			struct {
				id       string
				category string
				terms    []string
				class    FindingClass
				claim    string
				reason   string
				action   string
			}{"SDD-BOUNDARY", "architecture", []string{"boundary", "component", "service", "system context", "data flow", "interface"}, FindingBlocker, "System boundaries or interactions are not explicit.", "Implementation planning needs enough context to know what changes and what remains outside the design.", "Describe the affected boundary and important interactions."},
			struct {
				id       string
				category string
				terms    []string
				class    FindingClass
				claim    string
				reason   string
				action   string
			}{"SDD-DECISION", "decisions", []string{"decision", "rationale", "because", "trade-off", "alternative"}, FindingBlocker, "The design does not explain a material technical decision.", "An SDD should resolve technical choices, not only describe components.", "State the important choice and why it fits this context."},
			struct {
				id       string
				category string
				terms    []string
				class    FindingClass
				claim    string
				reason   string
				action   string
			}{"SDD-VERIFY", "verification", []string{"verify", "test", "validation", "evidence", "acceptance"}, FindingSuggestion, "The design has no visible verification approach.", "Material design claims should have a way to prove or inspect them.", "Map important requirements or invariants to verification."},
		)
	}

	for _, check := range checks {
		found := false
		for _, term := range check.terms {
			if strings.Contains(plain, term) {
				found = true
				break
			}
		}
		if !found {
			findings = append(findings, Finding{
				ID: check.id, Class: check.class, Category: check.category,
				Claim: check.claim, Reason: check.reason, SuggestedAction: check.action,
				Source: "deterministic",
			})
		}
	}

	return findings
}

func placeholderFindings(doc parsedDocument) []Finding {
	var findings []Finding
	inFence := false
	for index, line := range doc.lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if placeholderPattern.MatchString(line) {
			findings = append(findings, Finding{
				ID: "DOC-PLACEHOLDER", Class: FindingDecision, Category: "completeness", Line: index + 1,
				Claim:           "The document contains an unresolved placeholder.",
				Reason:          "A placeholder can hide a material unresolved decision.",
				SuggestedAction: "Resolve it, convert it to an explicit assumption, or record it as a known open question.",
				Source:          "deterministic",
			})
		}
	}
	return findings
}

func nfrFindings(doc parsedDocument) []Finding {
	var findings []Finding
	inFence := false
	for index, line := range doc.lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if genericNFRPattern.MatchString(line) {
			findings = append(findings, Finding{
				ID: "DOC-CONTEXTLESS-QUALITY", Class: FindingSuggestion, Category: "quality-attribute", Line: index + 1,
				Claim:           "A quality attribute is stated without enough context to guide a decision or verification.",
				Reason:          "Words such as scalable, secure, reliable, or fast do not define useful behavior by themselves.",
				SuggestedAction: "Add context, rationale, and a verification target when they are material. Otherwise remove the statement.",
				Source:          "deterministic",
			})
		}
	}
	return findings
}
