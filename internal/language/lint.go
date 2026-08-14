package language

import (
	"regexp"
	"strings"
	"unicode"
)

type Options struct {
	ProceduralSentenceMaxWords  int
	DescriptiveSentenceMaxWords int
}

func DefaultOptions() Options {
	return Options{ProceduralSentenceMaxWords: 20, DescriptiveSentenceMaxWords: 25}
}

type Finding struct {
	ID              string `json:"id"`
	Class           string `json:"class"`
	Category        string `json:"category"`
	Line            int    `json:"line,omitempty"`
	Claim           string `json:"claim"`
	Reason          string `json:"reason"`
	SuggestedAction string `json:"suggested_action,omitempty"`
	Source          string `json:"source"`
}

var sentencePattern = regexp.MustCompile(`[^.!?]+[.!?]?`)
var contractionPattern = regexp.MustCompile(`(?i)\b(can't|cannot've|won't|don't|doesn't|didn't|isn't|aren't|wasn't|weren't|shouldn't|wouldn't|couldn't|it's|that's|there's|they're|we're|you're|i'm|we'll|you'll|they'll|we've|you've|they've)\b`)
var imperativeStarts = map[string]struct{}{
	"add": {}, "apply": {}, "build": {}, "check": {}, "click": {}, "configure": {}, "create": {}, "delete": {},
	"deploy": {}, "disable": {}, "enable": {}, "ensure": {}, "enter": {}, "install": {}, "open": {}, "record": {},
	"remove": {}, "review": {}, "run": {}, "select": {}, "set": {}, "start": {}, "stop": {}, "update": {}, "use": {}, "verify": {},
}

func Lint(content string, glossary Glossary, options Options) []Finding {
	if options.ProceduralSentenceMaxWords <= 0 {
		options.ProceduralSentenceMaxWords = 20
	}
	if options.DescriptiveSentenceMaxWords <= 0 {
		options.DescriptiveSentenceMaxWords = 25
	}

	var findings []Finding
	lines := strings.Split(content, "\n")
	inFence := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "|") {
			continue
		}

		clean := strings.TrimSpace(strings.TrimLeft(trimmed, "-*+0123456789. "))
		for _, sentence := range sentencePattern.FindAllString(clean, -1) {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}
			words := wordCount(sentence)
			limit := options.DescriptiveSentenceMaxWords
			kind := "descriptive"
			if isProcedural(sentence, glossary) {
				limit = options.ProceduralSentenceMaxWords
				kind = "procedural"
			}
			if words > limit {
				findings = append(findings, Finding{
					ID: "STE-SENTENCE-LENGTH", Class: "suggestion", Category: "language", Line: index + 1,
					Claim:           "A " + kind + " sentence exceeds the STE-assisted word limit.",
					Reason:          "Shorter technical sentences reduce ambiguity and make agent instructions easier to verify.",
					SuggestedAction: "Split the sentence without changing its technical meaning.", Source: "ste-assisted",
				})
			}
			if contractionPattern.MatchString(sentence) {
				findings = append(findings, Finding{
					ID: "STE-CONTRACTION", Class: "nit", Category: "language", Line: index + 1,
					Claim:           "The sentence contains a contraction.",
					Reason:          "The STE-assisted profile avoids contractions in technical prose.",
					SuggestedAction: "Use the full form when this is generated technical prose.", Source: "ste-assisted",
				})
			}
		}
	}
	return findings
}

func wordCount(value string) int {
	return len(strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",;:()[]{}<>\"", r)
	}))
}

func isProcedural(sentence string, glossary Glossary) bool {
	fields := strings.Fields(strings.ToLower(sentence))
	if len(fields) == 0 {
		return false
	}
	first := strings.Trim(fields[0], "`*_()[]{}:;,.!?\"")
	if _, ok := imperativeStarts[first]; ok {
		return true
	}
	for _, verb := range glossary.TechnicalVerbs {
		if strings.EqualFold(first, verb) {
			return true
		}
	}
	return false
}
